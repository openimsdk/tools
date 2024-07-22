package memamq

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

//func TestNewMemoryQueue(t *testing.T) {
//	workerCount := 3
//	bufferSize := 10
//	queue := NewMemoryQueue(workerCount, bufferSize)
//
//	if cap(queue.taskChan) != bufferSize {
//		t.Errorf("Expected buffer size %d, got %d", bufferSize, cap(queue.taskChan))
//	}
//
//	if queue.isStopped {
//		t.Errorf("New queue is prematurely stopped")
//	}
//
//	if len(queue.taskChan) != 0 {
//		t.Errorf("New queue should be empty, found %d items", len(queue.taskChan))
//	}
//}

func TestPushAndStop(t *testing.T) {
	queue := NewMemoryQueue(1, 5)

	var wg sync.WaitGroup
	wg.Add(1)
	queue.Push(func() {
		time.Sleep(50 * time.Millisecond) // Simulate task delay
		wg.Done()
	})

	queue.Stop()
	wg.Wait()

	if err := queue.Push(func() {}); err == nil {
		t.Error("Expected error when pushing to stopped queue, got none")
	}
}

func TestPushTimeout(t *testing.T) {
	queue := NewMemoryQueue(1, 1) // Small buffer and worker to force full queue

	done := make(chan bool)
	go func() {
		queue.Push(func() {
			time.Sleep(200 * time.Millisecond) // Long enough to cause the second push to timeout
		})
		done <- true
	}()

	<-done // Ensure first task is pushed

	if err := queue.Push(func() {}); err != nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestName(t *testing.T) {
	queue := NewMemoryQueue(16, 1024)
	var count atomic.Int64
	for i := 0; i < 128; i++ {
		go func() {
			for {
				queue.Push(func() {
					count.Add(1)
				})
			}
		}()
	}

	<-time.After(time.Second * 2)
	t.Log("stop 1", time.Now())
	queue.Stop()
	t.Log("stop 2", time.Now())
	t.Log(count.Load(), time.Now())
}
