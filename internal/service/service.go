package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	ext_mongo "github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/extensions/ext_mongo"
	ext_redis "github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/extensions/ext_redis"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/proto_gens"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/common"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/jwt"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/mq"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/utils"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/utils/gopool"
)

func ServeWebsocketConnection(w http.ResponseWriter, r *http.Request) {
	traceId, spanId := uuid.New().String(), uuid.New().String()
	if v := r.Header.Get(common.ReqHeaderKeyRequestId); len(v) > 0 {
		traceId = v
	}
	_logger := logger.GetGlobalLogger().
		WithField(common.LoggerKeyTraceId, traceId).
		WithField(common.LoggerKeySpanId, spanId)
	_logger.Infof("New connection accessed. ClientIP:%s.", r.RemoteAddr)
	ctx := context.WithValue(context.WithValue(context.Background(), common.ContextKeyTraceId, traceId), common.ContextKeySpanId, spanId)

	var uid string
	if v := r.Header.Get(common.ReqHeaderKeyUid); len(v) > 0 {
		uid = v
	} else {
		_logger.Errorf("Invalid connection, no '%s' header carried.",
			common.ReqHeaderKeyUid)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	_logger = _logger.WithField(common.LoggerKeyUid, uid)

	if strings.HasPrefix(uid, "LT") {
		_logger.Debugf("Load test connection, uid:%s", uid)
	} else {
		var account, token string
		if v := r.Header.Get(common.ReqHeaderKeyAccount); len(v) > 0 {
			account = v
		} else {
			_logger.Errorf("Invalid connection, no '%s' header carried.",
				common.ReqHeaderKeyAccount)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if v := r.Header.Get(common.ReqHeaderKeyToken); len(v) > 0 {
			token = v
		} else {
			_logger.Errorf("Invalid connection, no '%s' header carried.",
				common.ReqHeaderKeyToken)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		valid, err := jwt.GetJWTManager().VerifyAccessToken(ctx, account, token)
		if err != nil {
			_logger.WithError(err).Error("Internal Server Error")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !valid {
			_logger.Error("Unauthorized connection.")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		// NOTE: 检查App版本, 如果不是支持的版本, 则拒绝连接.
		skip := os.Getenv("SKIP_APP_VERSION_CHECK") == "true"
		if account == "18033060554" {
			// NOTE: 18033060554是超级管理员“尾野酱”, 她不受任何限制.
			skip = true
		} else if strings.HasPrefix(account, "111222333") && (account >= "11122233301" && account <= "11122233305") {
			skip = true
		} else if strings.HasPrefix(account, "111222333") && (account >= "11122233395" && account <= "11122233398") {
			skip = true
		}
		if !skip {
			version := os.Getenv("SUPPORTED_MAJOR_AND_MINOR_APP_VERSION")

			var appVer string
			if v := r.Header.Get(common.ReqHeaderKeyAppVersion); len(v) > 0 {
				appVer = v
			} else {
				_logger.Warningf("Invalid connection, no '%s' header carried.",
					common.ReqHeaderKeyAppVersion)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if len(appVer) == 0 {
				_logger.Warningf("Invalid connection, '%s' header is empty.",
					common.ReqHeaderKeyAppVersion)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if !utils.CheckApiVersion(appVer) {
				_logger.Warningf("Invalid connection, '%s' header is in wrong format.",
					common.ReqHeaderKeyAppVersion)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if strings.Join(strings.Split(appVer, ".")[0:2], ".") != version {
				logger.GetGlobalLogger().Warningf("非法请求, uid:%s, old_app_version:%s, curr_major_and_minor_app_version:%s",
					uid, appVer, version)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
	}

	conn, err := _GW.upgrader.Upgrade(w, r, nil)
	if err != nil {
		_logger.WithError(err).Error("Failed to upgrade.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	conn.SetReadLimit(1024 * 1024 * 4)
	_logger.Info("New connection has been hosting.")
	if err = _GW.AddConn(uid, conn, ClientIP(r)); err != nil {
		_logger.WithError(err).Error("Failed to upgrade.")
		http.Error(w, http.StatusText(http.StatusInsufficientStorage), http.StatusInsufficientStorage)
		return
	}

	env := config.GetConfig().DeploymentEnv

	// NOTE: 执行框架层的心跳检测, C -> S
	conn.SetPingHandler(func(message string) error {
		_conn := _GW.GetConn(uid)
		if _conn == nil {
			return nil
		}

		_conn.mu.Lock()
		defer _conn.mu.Unlock()
		if err := _conn.conn.WriteMessage(websocket.PongMessage, []byte("Success")); err == nil {
			// NOTE: 用于验证TCP半关闭的情况.
			_logger.Trace("Received ping message from client.")
		} else {
			_logger.WithError(err).Warning("Failed to write pong message into conn for user.")
		}
		return err
	})
	// NOTE: 执行业务层的心跳检测, S -> C
	gopool.Go(func() {
		retries := 0
		for {
			_conn := _GW.GetConn(uid)
			if _conn == nil {
				break
			}

			time.Sleep(15 * time.Second)

			_conn.mu.Lock()
			message, _ := proto.Marshal(&proto_gens.MessageInfo{
				MessageType:      proto_gens.MessageType_MESSAGE_TYPE_KEEPALIVE_PING,
				MessageBody:      nil,
				DeliveryRequired: false,
			})
			err := _conn.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				retries++
				_logger.WithError(err).Warning("Failed to write ping message into conn for user.")
			} else {
				_logger.Trace("Write ping message into conn for user.")
			}
			_conn.mu.Unlock()
			if err != nil && retries >= 3 {
				// NOTE: 标记用户离线
				cacheItem := &ext_redis.Item{}
				cacheItem.SetKey(fmt.Sprintf("gcp_ags_%s_user_%s_online", env, uid))
				cacheItem.SetValue("0")
				cacheItem.SetTTL(0)
				_ = ext_redis.GetConnPool().GetBigCache().SetString(context.Background(), cacheItem)
				// 给biz层发送事件, 让biz层来处理好友即时下线状态刷新
				message, _ := proto.Marshal(&proto_gens.MessageInfo{
					TraceId:     uuid.New().String(),
					MessageType: proto_gens.MessageType_MESSAGE_TYPE_USER_ONLINE_STATUS_OFFLINE,
					MessageBody: nil,
				})
				mq.SendMessage(uid, message)
				_logger.Warning("Marked user offline (we cannot send keepalive message to client).")
				break
			}
		}
	})

	// NOTE: 标记用户在线
	cacheItem := &ext_redis.Item{}
	cacheItem.SetKey(fmt.Sprintf("gcp_ags_%s_user_%s_online", env, uid))
	cacheItem.SetValue("1")
	cacheItem.SetTTL(0)
	if err = ext_redis.GetConnPool().GetBigCache().SetString(ctx, cacheItem); err != nil {
		_logger.WithError(err).Error("Failed to set user online.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		_logger.Debug("Set user online.")
		// NOTE: 记录用户上线时间
		cacheItem = &ext_redis.Item{}
		cacheItem.SetKey(fmt.Sprintf("gcp_ags_%s_user_%s_online_ts", env, uid))
		cacheItem.SetValue(time.Now().Unix())
		cacheItem.SetTTL(0)
		_ = ext_redis.GetConnPool().GetBigCache().SetInt64(ctx, cacheItem)
		// NOTE: 好友上线通知
		gopool.Go(func() {
			var lastOnlineNotifyTime int64
			key := fmt.Sprintf("gcp_ags_%s_user_%s_online_notify_ts", env, uid)
			_ = ext_redis.GetConnPool().GetBigCache().GetInt64(context.Background(), key, &lastOnlineNotifyTime)
			now := time.Now().Unix()
			if lastOnlineNotifyTime == 0 || now-lastOnlineNotifyTime > 600 {
				// NOTE: 5分钟内只通知一次
				// 给biz层发送事件, 让biz层来处理好友上线通知
				message, _ := proto.Marshal(&proto_gens.MessageInfo{
					TraceId:     traceId,
					MessageType: proto_gens.MessageType_MESSAGE_TYPE_USER_ONLINE,
					MessageBody: nil,
				})
				mq.SendMessage(uid, message)
				// 给biz层发送事件, 让biz层来处理好友即时上线状态刷新
				message, _ = proto.Marshal(&proto_gens.MessageInfo{
					TraceId:     traceId,
					MessageType: proto_gens.MessageType_MESSAGE_TYPE_USER_ONLINE_STATUS_ONLINE,
					MessageBody: nil,
				})
				mq.SendMessage(uid, message)
				// 记录通知时间
				cacheItem := &ext_redis.Item{}
				cacheItem.SetKey(key)
				cacheItem.SetValue(now)
				cacheItem.SetTTL(0)
				_ = ext_redis.GetConnPool().GetBigCache().SetInt64(context.Background(), cacheItem)
			}
		})
	}
	// NOTE: 注册用户的长连接
	cacheItem = &ext_redis.Item{}
	cacheItem.SetKey(fmt.Sprintf("gcp_ags_%s_user_%s_conn", env, uid))
	// NOTE: 这里记录建连的主机名, 用于后续的消息推送
	cacheItem.SetValue(utils.GetHostname())
	cacheItem.SetTTL(0)
	if err = ext_redis.GetConnPool().GetBigCache().SetString(ctx, cacheItem); err != nil {
		_logger.WithError(err).Error("Failed to set user conn.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		_logger.Debug("Set user conn.")
	}
	// NOTE: 通过邮箱模型来保证消息触达, 为用户重连后做在断线期间遗漏的重要消息的重推
	gopool.Go(func() {
		conn := _GW.GetConn(uid)
		if conn == nil {
			return
		}

		var cnt int
		key := fmt.Sprintf("gcp_ags_%s_user_%s_offline_message_cnt", env, uid)
		_ = ext_redis.GetConnPool().GetBigCache().GetInt(context.Background(), key, &cnt)
		if cnt > 0 {
			messages := make([]string, 0, cnt)

			offset := 0
			limit := 10
			loaded := 0
			for loaded < cnt {
				// 分批捞出用户的离线消息
				if offset+limit > cnt {
					limit = cnt - offset
				}

				opt := &ext_mongo.FilterOption{}
				opt.SetUid(uid)
				opt.SetPageOffset(int32(offset))
				opt.SetPageLimit(int32(limit))
				pMessages, err := ext_mongo.GetConnPool().ListUnreadOfflineDeliveryRequiredMessages(context.Background(), opt)
				if err != nil {
					_logger.WithError(err).Error("Failed to list unread OfflineDeliveryRequiredMessages.")
					break
				}
				messages = append(messages, pMessages...)

				loaded += limit
			}

			if len(messages) > 0 {
				// 捞出的消息按照时间戳倒序排列, 但是要按照时间戳正序推送
				sort.Sort(sort.Reverse(sort.StringSlice(messages)))

				for _, message := range messages {
					// NOTE: github.com/gorilla/websocket 提供的写接口不是并发安全的
					conn.mu.Lock()
					if err := conn.conn.WriteMessage(websocket.BinaryMessage, utils.String2ByteSlice(message)); err != nil {
						_logger.WithError(err).Error("Failed to write one OfflineDeliveryRequiredMessage into conn for user.")
					} else {
						_logger.Trace("Write one OfflineDeliveryRequiredMessage into conn for user.")
					}
					conn.mu.Unlock()
				}
			}

			// NOTE: 不管重推成功与否, 都要清空用户的离线消息计数器
			cacheItem := &ext_redis.Item{}
			cacheItem.SetKey(key)
			cacheItem.SetValue(0)
			cacheItem.SetTTL(0)
			_ = ext_redis.GetConnPool().GetBigCache().SetInt(context.Background(), cacheItem)
		}
	})
	// 读取来自客户端的消息
	gopool.Go(func() {
		ServeWebsocketMessage(uid)
	})

	if strings.HasPrefix(uid, "LT") {
		utils.PrintMemUsage()
	}
}
