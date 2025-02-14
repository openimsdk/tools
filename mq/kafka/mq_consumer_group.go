package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mq"
)

func NewMConsumerGroupV2(ctx context.Context, conf *Config, groupID string, topics []string, autoCommitEnable bool) (mq.Consumer, error) {
	config, err := BuildConsumerGroupConfig(conf, sarama.OffsetNewest, autoCommitEnable)
	if err != nil {
		return nil, err
	}
	group, err := NewConsumerGroup(config, conf.Addr, groupID)
	if err != nil {
		return nil, err
	}
	mcg := &mqConsumerGroup{
		topics:   topics,
		groupID:  groupID,
		consumer: group,
		session:  make(chan *sessionClaim, 1),
		idle:     make(chan struct{}, 1),
	}
	mcg.idle <- struct{}{}
	mcg.loopConsume()
	return mcg, nil
}

type sessionClaim struct {
	session sarama.ConsumerGroupSession
	claim   sarama.ConsumerGroupClaim
}

type mqConsumerGroup struct {
	topics   []string
	groupID  string
	consumer sarama.ConsumerGroup
	cancel   context.CancelFunc
	idle     chan struct{}
	session  chan *sessionClaim
	curr     *sessionClaim
}

func (*mqConsumerGroup) Setup(sarama.ConsumerGroupSession) error { return nil }

func (*mqConsumerGroup) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (x *mqConsumerGroup) loopConsume() {
	var ctx context.Context
	ctx, x.cancel = context.WithCancel(context.Background())
	go func() {
		defer func() {
			close(x.session)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case x.idle <- struct{}{}:
			}
			if err := x.consumer.Consume(ctx, x.topics, x); err != nil {
				log.ZWarn(ctx, "consume err", err, "topic", x.topics, "groupID", x.groupID)
			}
		}
	}()
}

func (x *mqConsumerGroup) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	x.session <- &sessionClaim{session, claim}
	return nil
}

func (x *mqConsumerGroup) Subscribe(ctx context.Context, fn mq.Handler) error {
	for {
		curr := x.curr
		if curr == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case val, ok := <-x.session:
				if !ok {
					return sarama.ErrClosedConsumerGroup
				}
				curr = val
				x.curr = val
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case val, ok := <-curr.claim.Messages():
			if !ok {
				x.curr = nil
				select {
				case <-x.idle:
				default:
				}
				continue
			}
			ctx := GetContextWithMQHeader(val.Headers)
			if err := fn(ctx, string(val.Key), val.Value); err != nil {
				return err
			}
			curr.session.MarkMessage(val, "")
			curr.session.Commit()
		}
	}
}

func (x *mqConsumerGroup) Close() error {
	x.cancel()
	return x.consumer.Close()
}
