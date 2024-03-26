package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/errs"
)

func CheckKafka(ctx context.Context, conf Config) error {
	kfk, err := BuildConsumerGroupConfig(conf, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	cli, err := sarama.NewClient(conf.Addr, kfk)
	if err != nil {
		return errs.WrapMsg(err, "NewClient failed", "addr", conf.Addr, "config", *kfk)
	}
	if err := cli.Close(); err != nil {
		return errs.WrapMsg(err, "Close failed")
	}
	return nil
}
