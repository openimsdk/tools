package sre

import (
	"math/rand"
	"testing"
	"time"

	"github.com/openimsdk/tools/stability/internal/window"
	"github.com/stretchr/testify/assert"
)

func getBreaker() *sreBreaker {
	opt := options{
		success: 0.6,
		request: 100,
		bucket:  10,
		window:  3 * time.Second,
	}

	counterOpt := window.RollingCounterOpts{
		Size:           10,
		BucketDuration: time.Millisecond * 100,
	}

	stat := window.NewRollingCounter(counterOpt)

	return &sreBreaker{
		stat:    stat,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		request: opt.request,
		state:   StateClosed,
		k:       1 / opt.success,
	}
}

func TestSRE(t *testing.T) {
	b := getBreaker()
	testSREClose(t, b)

	b = getBreaker()
	testSREOpen(t, b)

	b = getBreaker()
	testSREHalfOpen(t, b)
}

func TestCallState(t *testing.T) {
	b := getBreaker()
	breakerMarkSuccess(b, 80)
	assert.Equal(t, nil, b.Allow())
	assert.Equal(t, StateClosed, b.state)

	breakerMarkFailed(b, 12000000)
	assert.NotEqual(t, nil, b.Allow())
	assert.Equal(t, StateOpen, b.state)
}

func breakerMarkSuccess(b *sreBreaker, count int) {
	for i := 0; i < count; i++ {
		b.MarkSuccess()
	}
}

func breakerMarkFailed(b *sreBreaker, count int) {
	for i := 0; i < count; i++ {
		b.MarkFailed()
	}
}

func testSREClose(t *testing.T, b *sreBreaker) {
	breakerMarkSuccess(b, 80)
	assert.Equal(t, nil, b.Allow())
	breakerMarkSuccess(b, 120)
	assert.Equal(t, nil, b.Allow())
}

func testSREOpen(t *testing.T, b *sreBreaker) {
	breakerMarkSuccess(b, 100)
	assert.Equal(t, nil, b.Allow(), " should be closed")
	breakerMarkFailed(b, 10000000)
	assert.NotEqual(t, nil, b.Allow(), " should be open")
}

func testSREHalfOpen(t *testing.T, b *sreBreaker) {
	// failback
	assert.Equal(t, nil, b.Allow())
	t.Run("allow single failed", func(t *testing.T) {
		breakerMarkFailed(b, 10000000)
		assert.NotEqual(t, nil, b.Allow())
	})
	time.Sleep(2 * time.Second)
	t.Run("allow single succeed", func(t *testing.T) {
		assert.Equal(t, nil, b.Allow())
		breakerMarkSuccess(b, 10000000)
		assert.Equal(t, nil, b.Allow())
	})
}
