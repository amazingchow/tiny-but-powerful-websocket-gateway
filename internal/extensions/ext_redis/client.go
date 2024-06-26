package extredis

import (
	"context"
	"crypto/tls"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type RedisClientConnPool struct {
	logger *logrus.Entry

	timeout time.Duration
	client  *redis.Client

	bc *BigCache
}

var p *RedisClientConnPool

func InitConnPool(cfg *config.Cache) {
	var err error

	p = &RedisClientConnPool{
		logger: logger.GetGlobalLogger().WithField("infra", "redis"),
	}
	if cfg.ConnTimeout > 0 {
		p.timeout = time.Duration(cfg.ConnTimeout) * time.Second
	} else {
		p.timeout = 5 * time.Second
	}

	ctx, cancel := NewConnPoolContext(context.Background())
	defer cancel()

	var tlsConfig *tls.Config
	if cfg.EnableSSL {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		tlsConfig = nil
	}
	// More info: https://redis.uptrace.dev/guide/go-redis-debugging.html
	rdb := redis.NewClient(&redis.Options{
		Network:               "tcp",
		Addr:                  cfg.Endpoint,
		ClientName:            "wechat-payment-callback-service",
		Username:              "",
		Password:              cfg.Pwd,
		DB:                    int(cfg.DB),
		ReadTimeout:           p.timeout,
		WriteTimeout:          p.timeout,
		ContextTimeoutEnabled: true,
		PoolFIFO:              false,
		PoolSize:              10, // 10 connections per every available CPU as reported by runtime.GOMAXPROCS
		PoolTimeout:           p.timeout + time.Second,
		MinIdleConns:          5,
		TLSConfig:             tlsConfig,
	})
	if _, err = rdb.Ping(ctx).Result(); err != nil {
		_ = rdb.Close()
		p.logger.WithError(err).Fatalf("Failed to connect to redis server using %s.",
			cfg.Endpoint)
	} else {
		p.logger.Debugf("Connected to redis server @\x1b[1;31m%s/%d\x1b[0m.",
			cfg.Endpoint, cfg.DB)
	}

	p.client = rdb
	p.bc = InstallBigCache(p)
}

func GetConnPool() *RedisClientConnPool {
	return p
}

func CloseConnPool() {
	if p != nil && p.client != nil {
		if err := p.client.Close(); err != nil {
			p.logger.WithError(err).Error("Failed to disconnect from redis server.")
		}
	}
}

func NewConnPoolContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, p.timeout)
}

func (p *RedisClientConnPool) GetBigCache() *BigCache {
	return p.bc
}
