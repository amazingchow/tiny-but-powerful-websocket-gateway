package extmongo

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type MongoClientConnPool struct {
	logger *logrus.Entry

	timeout time.Duration
	client  *mongo.Client

	database    *mongo.Database
	collections map[string]*mongo.Collection
}

var p *MongoClientConnPool

func InitConnPool(cfg *config.Storage) {
	var err error

	p = &MongoClientConnPool{
		logger: logger.GetGlobalLogger().WithField("infra", "mongo"),
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
	p.client, err = mongo.Connect(ctx,
		options.Client().ApplyURI(fmt.Sprintf("mongodb://%s/", cfg.Endpoint)).
			SetAuth(options.Credential{
				AuthMechanism: "SCRAM-SHA-256",
				AuthSource:    "admin",
				Username:      cfg.RootUsr,
				Password:      cfg.RootPwd,
			}).
			SetDirect(true).
			SetServerSelectionTimeout(60*time.Second).
			SetTimeout(p.timeout).
			SetTLSConfig(tlsConfig),
	)
	if err != nil {
		p.logger.WithError(err).Fatalf("Failed to connect to mongo server using mmongodb://%s/.",
			cfg.Endpoint)
	} else {
		p.logger.Debugf("Connected to mongo server @mongodb://%s/.",
			cfg.Endpoint)
	}

	if err = p.client.Ping(context.Background(), readpref.Primary()); err != nil {
		p.logger.WithError(err).Fatal("failed to ping mongo server")
	}

	p.database = p.client.Database(cfg.DB)
	p.collections = make(map[string]*mongo.Collection, 2)
	p.collections[PlatformOrderCollection] = p.database.Collection(PlatformOrderCollection)

	// 给 PlatformOrderCollection 创建额外的索引
	indexes := []string{"trade_id"}
	indexOrders := []int{1}
	for i := 0; i < len(indexes); i++ {
		index, err := p.collections[PlatformOrderCollection].Indexes().CreateOne(context.Background(), mongo.IndexModel{
			Keys: bson.D{
				{Key: indexes[i], Value: indexOrders[i]},
			},
			Options: options.Index().SetName(fmt.Sprintf("platform_order_%s_index", indexes[i])),
		})
		if err != nil {
			p.logger.WithError(err).Fatalf("failed to create index for %s.%s",
				cfg.DB, PlatformOrderCollection)
		} else {
			p.logger.Infof("create index %s for %s.%s",
				index, cfg.DB, PlatformOrderCollection)
		}
	}
}

func GetConnPool() *MongoClientConnPool {
	return p
}

func CloseConnPool() {
	if p != nil && p.client != nil {
		ctx, cancel := NewConnPoolContext(context.Background())
		defer cancel()
		if err := p.client.Disconnect(ctx); err != nil {
			p.logger.WithError(err).Error("Failed to disconnect from mongo server.")
		}
	}
}

func NewConnPoolContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, p.timeout)
}
