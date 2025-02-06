package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/mq"
)

func NewKafkaProducerV2(config *Config, addr []string, topic string) (mq.Producer, error) {
	conf, err := BuildProducerConfig(*config)
	if err != nil {
		return nil, err
	}
	producer, err := NewProducer(conf, addr)
	if err != nil {
		return nil, err
	}
	return &mqProducer{
		topic:    topic,
		producer: producer,
	}, nil
}

type mqProducer struct {
	topic    string
	producer sarama.SyncProducer
}

func (x *mqProducer) SendMessage(ctx context.Context, key string, value []byte) error {
	headers, err := GetMQHeaderWithContext(ctx)
	if err != nil {
		return err
	}
	kMsg := &sarama.ProducerMessage{
		Topic:   x.topic,
		Key:     sarama.StringEncoder(key),
		Value:   sarama.ByteEncoder(value),
		Headers: headers,
	}
	if _, _, err := x.producer.SendMessage(kMsg); err != nil {
		return err
	}
	return nil
}

func (x *mqProducer) Close() error {
	return x.producer.Close()
}
