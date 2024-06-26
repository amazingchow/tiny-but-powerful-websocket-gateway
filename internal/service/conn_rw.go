package service

import (
	"context"
	"fmt"
	"net"

	"github.com/gorilla/websocket"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	_ "github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/extensions/ext_mongo"
	ext_redis "github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/extensions/ext_redis"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/common"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/mq"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/utils"
)

func ServeWebsocketMessage(uid string) {
	conn := _GW.GetConn(uid)
	defer _GW.DelConn(uid)

	_logger := logger.GetGlobalLogger().WithField(common.LoggerKeyUid, uid)
	retries := 0
IO_LOOP:
	for {
		if conn == nil || conn.conn == nil {
			break IO_LOOP
		}
		// NOTE: github.com/gorilla/websocket 提供的读接口是并发安全的
		msgType, msg, err := conn.conn.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				if closeErr.Code == 1000 {
					_logger.Trace("Client closed conn.")
				} else {
					_logger.Warningf("Client abnormally closed conn, err-code: %d.", closeErr.Code)
				}
				break IO_LOOP
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				_logger.WithError(err).Error("Timeout to read from conn.")
				retries++
				if retries >= 3 {
					break IO_LOOP
				} else {
					continue
				}
			} else {
				_logger.WithError(err).Error("Failed to read from conn.")
				break IO_LOOP
			}
		}

		switch msgType {
		case websocket.CloseMessage:
			{
				_logger.Trace("Client closed conn.")
				break IO_LOOP
			}
		case websocket.BinaryMessage:
			{
				mq.SendMessage(uid, msg)
			}
		default:
			{
				_logger.Warnf("Message type '%d' not supported.", msgType)
				continue
			}
		}
	}

	// NOTE: 标记用户离线
	cacheItem := &ext_redis.Item{}
	cacheItem.SetKey(fmt.Sprintf("gcp_ags_%s_user_%s_online", config.GetConfig().DeploymentEnv, uid))
	cacheItem.SetValue("0")
	cacheItem.SetTTL(0)
	_ = ext_redis.GetConnPool().GetBigCache().SetString(context.Background(), cacheItem)
	_logger.Warning("Marked user offline (client disconnected normally).")
}

func MessageDispatcher(uid string, message []byte) {
	_logger := logger.GetGlobalLogger().WithField(common.LoggerKeyUid, uid)

	conn := _GW.GetConn(uid)
	if conn == nil {
		env := config.GetConfig().DeploymentEnv

		// NOTE: 当前节点没有该用户的连接, 说明:
		// 1）用户的连接在其他节点, 直接执行drop-it;
		// 2) 用户的连接已经断开, 需要通过邮箱模型来保证消息触达, 即需要持久化用户在断线期间遗漏的重要消息, 为其重连后做重推.
		var node string
		key := fmt.Sprintf("gcp_ags_%s_user_%s_conn", env, uid)
		_ = ext_redis.GetConnPool().GetBigCache().GetString(context.Background(), key, &node)
		if node == utils.GetHostname() {
			// 执行情况2)
			_logger.Warning("Failed to write one message into conn for user, since the conn has dropped.")

			var online string
			key := fmt.Sprintf("gcp_ags_%s_user_%s_online", env, uid)
			_ = ext_redis.GetConnPool().GetBigCache().GetString(context.Background(), key, &online)
			if online == "1" {
				// NOTE: 标记用户离线
				cacheItem := &ext_redis.Item{}
				cacheItem.SetKey(key)
				cacheItem.SetValue("0")
				cacheItem.SetTTL(0)
				_ = ext_redis.GetConnPool().GetBigCache().SetString(context.Background(), cacheItem)
				_logger.Warning("Marked user offline (client disconnected abnormally).")
			}

			// // NOTE: 持久化用户在断线期间遗漏的重要消息
			// var base types.MessageInfo
			// proto.Unmarshal(message, &base)
			// if base.DeliveryRequired {
			// 	if err := ext_mongo.GetConnPool().AddOfflineDeliveryRequiredMessage(
			// 		context.Background(), uid, utils.ByteSlice2String(message), base.DeliveryRequiredTtlInSecs); err == nil {
			// 		// NOTE: 递增用户的离线消息计数器, redis允许对一个不存在的key执行incr操作, 会把该key的值当作0来处理
			// 		key := fmt.Sprintf("gcp_ags_%s_user_%s_offline_message_cnt", env, uid)
			// 		_ = ext_redis.GetConnPool().GetBigCache().Incr(context.Background(), key)
			// 	}
			// }
		} else {
			// 执行情况2)
			_logger.Warningf("No need to write the message into conn for user, since the conn is in other node. curr_node:%s, other_node:%s.",
				utils.GetHostname(), node)
		}

		return
	}

	// NOTE: github.com/gorilla/websocket 提供的写接口不是并发安全的
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if err := conn.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		_logger.WithError(err).Error("Failed to write one message into conn for user.")
	} else {
		_logger.Trace("Write one message into conn for user.")
	}
}
