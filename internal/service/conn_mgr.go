package service

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/metrics"
)

type WebsocketConn struct {
	// NOTE: github.com/gorilla/websocket 提供的写接口并不是并发安全的
	mu       sync.Mutex
	conn     *websocket.Conn
	uid      string
	clientIP string
}

func (conn *WebsocketConn) GetUid() string {
	return conn.uid
}

func (conn *WebsocketConn) GetClientIP() string {
	return conn.clientIP
}

type WebsocketGateway struct {
	upgrader   *websocket.Upgrader
	mu         sync.RWMutex
	conns      map[string]*WebsocketConn // uid <-> conn
	connCnt    int32
	maxConnCnt int32
}

var _GW *WebsocketGateway

func SetUpWebsocketGateway(maxConn int32) {
	_GW = &WebsocketGateway{}
	_GW.upgrader = &websocket.Upgrader{
		HandshakeTimeout: 15 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
	}
	_GW.conns = make(map[string]*WebsocketConn)
	_GW.connCnt = 0
	_GW.maxConnCnt = maxConn

	go func() {
	RECONCILE_LOOP:
		for {
			time.Sleep(10 * time.Minute)
			if _GW.conns == nil {
				break RECONCILE_LOOP
			}
			_GW.ReconcileTotalConnCnt()
		}
	}()
}

func CloseWebsocketGateway() {
	_GW.mu.Lock()
	defer _GW.mu.Unlock()

	for _, conn := range _GW.conns {
		if conn != nil && conn.conn != nil {
			conn.conn.Close()
		}
	}
	_GW.conns = nil
}

func (gw *WebsocketGateway) ReconcileTotalConnCnt() {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	var cnt int32
	for _, conn := range gw.conns {
		if conn != nil && conn.conn != nil {
			cnt++
		}
	}
	gw.connCnt = cnt
}

func (gw *WebsocketGateway) AddConn(uid string, conn *websocket.Conn, clientIP string) (err error) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if gw.connCnt+1 > gw.maxConnCnt {
		conn.Close()
		logger.GetGlobalLogger().Warningf("Maximum connections exceeded. MaxConnLimit:%d.", gw.maxConnCnt)
		err = ErrExceedMaxConnNum
		return
	}

	gw.conns[uid] = &WebsocketConn{
		conn:     conn,
		uid:      uid,
		clientIP: clientIP,
	}

	gw.connCnt += 1
	metrics.WebsocketConnectionTotalCnt.WithLabelValues("infra-websocket-gateway-service").Inc()
	return
}

func (gw *WebsocketGateway) DelConn(uid string) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	gw.connCnt -= 1
	metrics.WebsocketConnectionTotalCnt.WithLabelValues("infra-websocket-gateway-service").Dec()

	conn, ok := gw.conns[uid]
	if !ok {
		return
	}
	if conn != nil && conn.conn != nil {
		conn.conn.Close()
	}
	gw.conns[uid] = nil
}

func (gw *WebsocketGateway) GetConn(uid string) (conn *WebsocketConn) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	var ok bool
	if conn, ok = gw.conns[uid]; ok {
		return
	}
	conn = nil
	return
}
