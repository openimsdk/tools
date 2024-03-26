package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/errs"
)

func CheckKafka(ctx context.Context, conf *Config, topics []string) error {
	kfk, err := BuildConsumerGroupConfig(conf, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	cli, err := sarama.NewClient(conf.Addr, kfk)
	if err != nil {
		return errs.WrapMsg(err, "NewClient failed", "addr", conf.Addr, "config", *kfk)
	}
	defer cli.Close()

	existingTopics, err := cli.Topics()
	if err != nil {
		return errs.WrapMsg(err, "Failed to list topics")
	}

	existingTopicsMap := make(map[string]bool)
	for _, t := range existingTopics {
		existingTopicsMap[t] = true
	}

	for _, topic := range topics {
		if !existingTopicsMap[topic] {
			return errs.New("topic not exist").WrapMsg("topic not exist", "topic", topic)
		}
	}
	return nil
}
