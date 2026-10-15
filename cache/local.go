package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type localEntry struct {
	value     []byte
	expiresAt time.Time
}

type localInflightCall struct {
	done    chan struct{}
	payload []byte
	hit     bool
	err     error
}

// LocalStore is an in-memory cache backend.
type LocalStore struct {
	mu         sync.RWMutex
	items      map[string]localEntry
	inflightMu sync.Mutex
	inflight   map[string]*localInflightCall
}

func NewLocalStore() *LocalStore {
	return &LocalStore{
		items:    make(map[string]localEntry),
		inflight: make(map[string]*localInflightCall),
	}
}

func (s *LocalStore) Backend() string { return "local" }

func (s *LocalStore) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	s.mu.RLock()
	entry, ok := s.items[key]
	s.mu.RUnlock()

	if !ok {
		RecordMiss()
		RecordTrace(ctx, s.Backend(), key, false, true)
		return false, nil
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.items, key)
		s.mu.Unlock()
		RecordMiss()
		RecordTrace(ctx, s.Backend(), key, false, true)
		return false, nil
	}

	if err := json.Unmarshal(entry.value, dest); err != nil {
		return false, err
	}

	RecordHit()
	RecordTrace(ctx, s.Backend(), key, true, true)
	return true, nil
}

func (s *LocalStore) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, loader func() (interface{}, error)) (bool, error) {
	found, err := s.Get(ctx, key, dest)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	payload, hit, err := s.doInflightLoad(key, func() ([]byte, bool, error) {
		s.mu.RLock()
		entry, ok := s.items[key]
		s.mu.RUnlock()

		if ok && (entry.expiresAt.IsZero() || time.Now().Before(entry.expiresAt)) {
			RecordHit()
			return entry.value, true, nil
		}

		loaded, loadErr := loader()
		if loadErr != nil {
			return nil, false, loadErr
		}

		payload, marshalErr := json.Marshal(loaded)
		if marshalErr != nil {
			return nil, false, marshalErr
		}

		entry = localEntry{value: payload}
		if ttl > 0 {
			entry.expiresAt = time.Now().Add(ttl)
		}

		s.mu.Lock()
		s.items[key] = entry
		s.mu.Unlock()

		RecordSet()
		return payload, false, nil
	})
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(payload, dest); err != nil {
		return false, err
	}

	RecordTrace(ctx, s.Backend(), key, hit, true)

	return hit, nil
}

func (s *LocalStore) doInflightLoad(key string, loader func() ([]byte, bool, error)) ([]byte, bool, error) {
	s.inflightMu.Lock()
	if call, exists := s.inflight[key]; exists {
		s.inflightMu.Unlock()
		<-call.done
		return call.payload, call.hit, call.err
	}

	call := &localInflightCall{done: make(chan struct{})}
	s.inflight[key] = call
	s.inflightMu.Unlock()

	call.payload, call.hit, call.err = loader()
	close(call.done)

	s.inflightMu.Lock()
	delete(s.inflight, key)
	s.inflightMu.Unlock()

	return call.payload, call.hit, call.err
}

func (s *LocalStore) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	entry := localEntry{value: payload}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	s.items[key] = entry
	s.mu.Unlock()
	RecordSet()

	return nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	RecordInvalidate()
	return nil
}

func (s *LocalStore) DeleteByPrefix(_ context.Context, prefix string) error {
	s.mu.Lock()
	for key := range s.items {
		if strings.HasPrefix(key, prefix) {
			delete(s.items, key)
			RecordInvalidate()
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *LocalStore) Close() error {
	s.mu.Lock()
	s.items = make(map[string]localEntry)
	s.mu.Unlock()
	return nil
}
