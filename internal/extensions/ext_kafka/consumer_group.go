package ext_kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type KafkaConsumerGroupConfig struct {
	Brokers         []string `json:"brokers"`
	EnableSSL       bool     `json:"enable_ssl"`
	EnableSASL      bool     `json:"enable_sasl"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	Topics          []string `json:"topics"`
	ConsumerGroupId string   `json:"consumer_group_id"`
	ClientId        string   `json:"client_id"`
	Version         string   `json:"version"`
	Concurrency     int      `json:"concurrency"`
	CacheSize       int      `json:"cache_size"`
}

type MessageConsumerGroup struct {
	logger *logrus.Entry

	ctx  context.Context
	cncl context.CancelFunc

	once sync.Once
	wg   sync.WaitGroup

	topics      []string
	cgroup      sarama.ConsumerGroup
	handler     *predatorImpl
	concurrency int
	messages    chan *sarama.ConsumerMessage
}

func NewMessageConsumerGroup(cfg *KafkaConsumerGroupConfig) (cg *MessageConsumerGroup, err error) {
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 1024
	}

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version, err = sarama.ParseKafkaVersion(cfg.Version)
	if err != nil {
		return
	}
	if len(os.Getenv("POD_NAME")) > 0 {
		// NOTE: 针对K8S部署场景, 为了区分不同的Pod实例, kafka客户端实例需要在ClientID中加入Pod的名称.
		kafkaCfg.ClientID = fmt.Sprintf("%s_%s", cfg.ClientId, os.Getenv("POD_NAME"))
	} else {
		kafkaCfg.ClientID = cfg.ClientId
	}
	if len(os.Getenv("POD_NAME")) > 0 {
		// NOTE: 针对K8S部署场景, 为了区分不同的Pod实例, kafka客户端实例需要在ClientID中加入Pod的名称.
		cfg.ConsumerGroupId = fmt.Sprintf("%s_%s", cfg.ConsumerGroupId, os.Getenv("POD_NAME"))
	}

	kafkaCfg.Net.ReadTimeout = 10 * time.Second
	if cfg.EnableSSL {
		kafkaCfg.Net.TLS.Enable = true
		kafkaCfg.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		kafkaCfg.Net.TLS.Enable = false
	}
	if cfg.EnableSASL {
		kafkaCfg.Net.SASL.Enable = true
		kafkaCfg.Net.SASL.User = cfg.Username
		kafkaCfg.Net.SASL.Password = cfg.Password
		kafkaCfg.Net.SASL.Version = sarama.SASLHandshakeV1
		kafkaCfg.Net.SASL.Handshake = true
		kafkaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		kafkaCfg.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
	} else {
		kafkaCfg.Net.SASL.Enable = false
	}
	kafkaCfg.Net.KeepAlive = 10 * time.Hour

	kafkaCfg.Consumer.Group.Session.Timeout = 10 * time.Second
	kafkaCfg.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	kafkaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	kafkaCfg.Consumer.MaxProcessingTime = 500 * time.Millisecond
	kafkaCfg.Consumer.Return.Errors = true
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	cg = &MessageConsumerGroup{
		logger:      logger.GetGlobalLogger().WithField("infra", "kafka-consumer-group"),
		wg:          sync.WaitGroup{},
		topics:      cfg.Topics,
		concurrency: cfg.Concurrency,
		messages:    make(chan *sarama.ConsumerMessage, cfg.CacheSize),
	}
	cg.cgroup, err = sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroupId, kafkaCfg)
	if err != nil {
		cg.logger.WithError(err).Error("Failed to setup new kafka consumer group.")
		return
	}
	cg.ctx, cg.cncl = context.WithCancel(context.Background())
	cg.handler = &predatorImpl{predator: cg, metadata: cfg.ClientId}
	cg.run()

	cg.logger.Infof("Setup new kafka consumer group <client-id: %s>.", cfg.ClientId)
	return
}

func (cg *MessageConsumerGroup) run() {
	cg.wg.Add(1)
	go func() {
	ERR_WATCH_LOOP:
		for {
			select {
			case err, ok := <-cg.cgroup.Errors():
				if !ok {
					break ERR_WATCH_LOOP
				}
				cg.logger.WithError(err).Errorf("Failed to consume from topic in one of %v.", cg.topics)
			case <-cg.ctx.Done():
				break ERR_WATCH_LOOP
			}
		}
		cg.wg.Done()
	}()

	for i := 0; i < cg.concurrency; i++ {
		cg.wg.Add(1)
		go cg.consumeMessages(i)
	}
}

func (cg *MessageConsumerGroup) Close() {
	cg.once.Do(func() {
		if cg.cncl != nil {
			cg.cncl()
		}
		close(cg.messages)
		cg.cgroup.Close()
		cg.wg.Wait()
		cg.logger.Info("Closed kafka consumer group.")
	})
}

func (cg *MessageConsumerGroup) Messages() chan *sarama.ConsumerMessage {
	return cg.messages
}

func (cg *MessageConsumerGroup) Topics() []string {
	return cg.topics
}

func (cg *MessageConsumerGroup) consumeMessages(id int) {
	defer cg.wg.Done()

CONSUMER_GROUP_LOOP:
	for {
		if err := cg.cgroup.Consume(cg.ctx, cg.topics, cg.handler); err != nil {
			if _err, ok := err.(net.Error); ok && _err.Timeout() {
				cg.logger.WithError(err).Warnf("Timeout to consume from topic in one of %v on consumer-goroutine-%d.", cg.topics, id)
			} else {
				cg.logger.WithError(err).Errorf("Failed to consume from topic in one of %v on consumer-goroutine-%d.", cg.topics, id)
			}
		}

		select {
		case <-cg.ctx.Done():
			break CONSUMER_GROUP_LOOP
		default:
			cg.logger.Debugf("Consumer group (topics: %v) rebalances on consumer-goroutine-%d.", cg.topics, id)
		}
	}
}

func (cg *MessageConsumerGroup) cacheMessage(msg *sarama.ConsumerMessage) {
	cg.messages <- msg
}

type predatorImpl struct {
	predator *MessageConsumerGroup
	metadata string
}

func (h *predatorImpl) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *predatorImpl) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *predatorImpl) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
CONSUME_LOOP:
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				break CONSUME_LOOP
			}
			h.predator.cacheMessage(msg)
			session.MarkMessage(msg, h.metadata)
		case <-h.predator.ctx.Done():
			break CONSUME_LOOP
		}
	}
	return nil
}
