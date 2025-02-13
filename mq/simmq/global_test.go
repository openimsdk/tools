package simmq

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestName(t *testing.T) {
	const topic = "test"
	ctx := context.Background()

	done := make(chan struct{})

	go func() {
		defer func() {
			t.Log("consumer end")
			close(done)
		}()
		fn := func(ctx context.Context, key string, value []byte) error {
			t.Logf("consumer key: %s, value: %s", key, value)
			return nil
		}
		c := GetTopicConsumer(topic)
		for {
			if err := c.Subscribe(ctx, fn); err != nil {
				t.Log("subscribe err", err)
				return
			}
		}
	}()

	var wg sync.WaitGroup

	var count atomic.Int64
	for i := 0; i < 4; i++ {
		wg.Add(1)
		p := GetTopicProducer(topic)
		key := fmt.Sprintf("go_%d", i+1)

		go func() {
			defer func() {
				t.Log("producer end", key)
				wg.Done()
			}()
			for i := 1; i <= 10; i++ {
				value := fmt.Sprintf("value_%d", count.Add(1))
				if err := p.SendMessage(ctx, key, []byte(value)); err != nil {
					t.Log("send message err", key, value, err)
					return
				}
			}
		}()
	}

	wg.Wait()
	_ = GetTopicProducer(topic).Close()
	<-done
}
