package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalStoreSetGetDelete(t *testing.T) {
	s := NewLocalStore()
	ctx := context.Background()

	if err := s.Set(ctx, "user:1", map[string]int{"n": 1}, time.Minute); err != nil {
		t.Fatal(err)
	}

	var got map[string]int
	found, err := s.Get(ctx, "user:1", &got)
	if err != nil || !found || got["n"] != 1 {
		t.Fatalf("Get = %v %v %v", got, found, err)
	}

	_ = s.DeleteByPrefix(ctx, "user:")
	if found, _ := s.Get(ctx, "user:1", &got); found {
		t.Fatal("expected key removed by prefix delete")
	}
}

func TestLocalStoreExpiry(t *testing.T) {
	s := NewLocalStore()
	ctx := context.Background()

	_ = s.Set(ctx, "k", "v", time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	var got string
	if found, _ := s.Get(ctx, "k", &got); found {
		t.Fatal("expected expired entry to miss")
	}
}

func TestLocalStoreGetOrSetCoalescesLoads(t *testing.T) {
	s := NewLocalStore()
	ctx := context.Background()

	var loads atomic.Int32
	loader := func() (interface{}, error) {
		loads.Add(1)
		time.Sleep(20 * time.Millisecond)
		return 7, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var v int
			if _, err := s.GetOrSet(ctx, "hot", &v, time.Minute, loader); err != nil || v != 7 {
				t.Errorf("GetOrSet = %d, %v", v, err)
			}
		}()
	}
	wg.Wait()

	if n := loads.Load(); n != 1 {
		t.Fatalf("loader ran %d times, want 1", n)
	}
}
