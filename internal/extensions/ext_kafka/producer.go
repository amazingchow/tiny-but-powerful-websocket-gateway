package ext_kafka

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type KafkaProducerConfig struct {
	Brokers         []string `json:"brokers"`
	EnableSSL       bool     `json:"enable_ssl"`
	EnableSASL      bool     `json:"enable_sasl"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	Topic           string   `json:"topic"`
	RetryTopic      string   `json:"retry_topic"`
	DeadLetterTopic string   `json:"dead_letter_topic"`
	ClientId        string   `json:"client_id"`
	Version         string   `json:"version"`
}

type MessageProducer struct {
	logger *logrus.Entry

	topic           string
	retryTopic      string
	deadLetterTopic string
	producer        sarama.AsyncProducer
}

func NewMessageProducer(cfg *KafkaProducerConfig) (p *MessageProducer, err error) {
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

	kafkaCfg.Net.WriteTimeout = 10 * time.Second
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

	// Only wait for the leader to ack.
	kafkaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaCfg.Producer.Compression = sarama.CompressionSnappy
	// Flush batches every 500ms.
	kafkaCfg.Producer.Flush.Frequency = 500 * time.Millisecond
	// Retry up to 5 times to produce the message.
	kafkaCfg.Producer.Retry.Max = 5
	kafkaCfg.Producer.Return.Successes = false
	kafkaCfg.Producer.Return.Errors = true

	p = &MessageProducer{
		logger:          logger.GetGlobalLogger().WithField("component", "kafka-producer"),
		topic:           cfg.Topic,
		retryTopic:      cfg.RetryTopic,
		deadLetterTopic: cfg.DeadLetterTopic,
	}
	p.producer, err = sarama.NewAsyncProducer(cfg.Brokers, kafkaCfg)
	if err != nil {
		p.logger.WithError(err).Error("Failed to setup new kafka producer.")
		return
	}

	p.logger.Infof("Setup new kafka producer <client-id: %s>.", cfg.ClientId)
	return
}

func (p *MessageProducer) Publish(bytes []byte) error {
	return p.publish("", "", -1, bytes)
}

func (p *MessageProducer) PublishByTopic(topic string, bytes []byte) error {
	return p.publish(topic, "", -1, bytes)
}

func (p *MessageProducer) PublishByKey(key string, bytes []byte) error {
	return p.publish("", key, -1, bytes)
}

func (p *MessageProducer) PublishByTopicAndKey(topic, key string, bytes []byte) error {
	return p.publish(topic, key, -1, bytes)
}

func (p *MessageProducer) publish(topic, key string, partition int32, bytes []byte) error {
	if p.producer == nil {
		p.logger.Error("Cannot publish message, kafka producer has been closed.")
		return nil
	}

	message := &sarama.ProducerMessage{
		Value: sarama.ByteEncoder(bytes),
	}
	if len(topic) > 0 {
		message.Topic = topic
	} else {
		message.Topic = p.topic
	}
	if len(key) > 0 {
		message.Key = sarama.StringEncoder(key)
	}
	if partition >= 0 {
		message.Partition = partition
	}

	select {
	case p.producer.Input() <- message:
		return nil
	case err := <-p.producer.Errors():
		p.logger.WithError(err).Error("Failed to publish message to brokers.")
		return err
	}
}

func (p *MessageProducer) Topic() string {
	return p.topic
}

func (p *MessageProducer) RetryTopic() string {
	return p.retryTopic
}

func (p *MessageProducer) DeadLetterTopic() string {
	return p.deadLetterTopic
}

func (p *MessageProducer) Close() {
	p.producer.Close()
	p.producer = nil
	p.logger.Info("Closed kafka producer.")
}
