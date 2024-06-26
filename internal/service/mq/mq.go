package mq

import (
	"fmt"
	"os"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	ext_kafka "github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/extensions/ext_kafka"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/utils/gopool"
)

var producer *ext_kafka.MessageProducer

func SetUpMessageQueueProducer(cfg *ext_kafka.KafkaProducerConfig) {
	var err error
	producer, err = ext_kafka.NewMessageProducer(cfg)
	if err != nil {
		logger.GetGlobalLogger().WithError(err).Fatal("Failed to create a new MessageQueueProducer.")
	}
}

func CloseMessageQueueProducer() {
	producer.Close()
}

func SendMessage(uid string, msg []byte) {
	if err := producer.PublishByKey(uid, msg); err != nil {
		logger.GetGlobalLogger().WithError(err).Errorf("Failed to send message from websocket conn <uid: %s> to topic:%s.", uid, producer.Topic())
	} else {
		logger.GetGlobalLogger().Tracef("Send message from websocket conn <uid: %s> to topic:%s.", uid, producer.Topic())
	}
}

var dispatcher func(key string, event []byte)

func RegisterMessageDispatcher(h func(key string, msg []byte)) {
	dispatcher = h
}

var consumergroup *ext_kafka.MessageConsumerGroup

func SetUpMessageQueueConsumerGroup(cfg *ext_kafka.KafkaConsumerGroupConfig) {
	var err error
	hostname, err := os.Hostname()
	if err != nil {
		logger.GetGlobalLogger().WithError(err).Fatal("Failed to get hostname.")
	}
	cfg.ConsumerGroupId = fmt.Sprintf("%s.%s", cfg.ConsumerGroupId, hostname)
	consumergroup, err = ext_kafka.NewMessageConsumerGroup(cfg)
	if err != nil {
		logger.GetGlobalLogger().WithError(err).Fatal("Failed to create a new MessageQueueConsumerGroup.")
	}
	go func() {
		logger.GetGlobalLogger().Infof("Ready to receive messages from topics:%v.", consumergroup.Topics())
		for msg := range consumergroup.Messages() {
			logger.GetGlobalLogger().Tracef("Receive a message from topic:%s.", msg.Topic)
			// 注意for-range循环中的变量可变性问题, 此处通过显式复制迭代变量的值来解决.
			k, v := msg.Key, msg.Value
			gopool.Go(func() {
				dispatcher(string(k), v)
			})
		}
	}()
}

func CloseMessageQueueConsumerGroup() {
	consumergroup.Close()
}
