package kafka

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mcontext"
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
		msg:      make(chan *consumerMessage, 64),
	}
	mcg.ctx, mcg.cancel = context.WithCancel(ctx)
	mcg.loopConsume()
	return mcg, nil
}

type consumerMessage struct {
	Msg     *sarama.ConsumerMessage
	Session sarama.ConsumerGroupSession
}

type mqConsumerGroup struct {
	topics   []string
	groupID  string
	consumer sarama.ConsumerGroup
	ctx      context.Context
	cancel   context.CancelFunc
	msg      chan *consumerMessage
	lock     sync.Mutex
}

func (*mqConsumerGroup) Setup(sarama.ConsumerGroupSession) error { return nil }

func (*mqConsumerGroup) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (x *mqConsumerGroup) closeMsgChan() {
	select {
	case <-x.ctx.Done():
		if x.lock.TryLock() {
			close(x.msg)
			x.lock.Unlock()
		}
	default:
	}
}

func (x *mqConsumerGroup) loopConsume() {
	go func() {
		defer x.closeMsgChan()
		ctx := mcontext.SetOperationID(x.ctx, fmt.Sprintf("consumer_group_%s_%s_%d", strings.Join(x.topics, "_"), x.groupID, rand.Uint32()))
		for {
			if err := x.consumer.Consume(x.ctx, x.topics, x); err != nil {
				switch {
				case errors.Is(err, context.Canceled):
					return
				case errors.Is(err, sarama.ErrClosedConsumerGroup):
					return
				}
				log.ZWarn(ctx, "consume err", err, "topic", x.topics, "groupID", x.groupID)
			}
		}
	}()
}

func (x *mqConsumerGroup) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	defer x.closeMsgChan()
	msg := claim.Messages()
	for {
		select {
		case <-x.ctx.Done():
			return context.Canceled
		case val, ok := <-msg:
			if !ok {
				return nil
			}
			x.msg <- &consumerMessage{Msg: val, Session: session}
		}
	}
}

func (x *mqConsumerGroup) Subscribe(ctx context.Context, fn mq.Handler) error {
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case msg, ok := <-x.msg:
		if !ok {
			return sarama.ErrClosedConsumerGroup
		}
		ctx := GetContextWithMQHeader(msg.Msg.Headers)
		if err := fn(ctx, string(msg.Msg.Key), msg.Msg.Value); err != nil {
			return err
		}
		msg.Session.MarkMessage(msg.Msg, "")
		msg.Session.Commit()
		return nil
	}
}

func (x *mqConsumerGroup) Close() error {
	x.cancel()
	return x.consumer.Close()
}
