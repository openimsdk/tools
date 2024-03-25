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
		return errs.WrapMsg(err, "addr", conf.Addr)
	}
	if err := cli.Close(); err != nil {
		return errs.WrapMsg(err, "close kafka")
	}
	return nil
}
