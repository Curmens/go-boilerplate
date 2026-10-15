package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"example.com/config"

	"github.com/redis/go-redis/v9"
)

type redisInflightCall struct {
	done    chan struct{}
	payload []byte
	hit     bool
	err     error
}

// RedisStore is a Redis-backed cache backend.
type RedisStore struct {
	client     *redis.Client
	inflightMu sync.Mutex
	inflight   map[string]*redisInflightCall
}

func NewRedisStore(cfg config.RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{client: client, inflight: make(map[string]*redisInflightCall)}, nil
}

func (s *RedisStore) Backend() string { return "redis" }

func (s *RedisStore) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	payload, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			RecordMiss()
			RecordTrace(ctx, s.Backend(), key, false, true)
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(payload, dest); err != nil {
		return false, err
	}

	RecordHit()
	RecordTrace(ctx, s.Backend(), key, true, true)
	return true, nil
}

func (s *RedisStore) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, loader func() (interface{}, error)) (bool, error) {
	found, err := s.Get(ctx, key, dest)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	payload, hit, err := s.doInflightLoad(key, func() ([]byte, bool, error) {
		payload, getErr := s.client.Get(ctx, key).Bytes()
		if getErr == nil {
			RecordHit()
			return payload, true, nil
		}
		if getErr != redis.Nil {
			return nil, false, getErr
		}

		loaded, loadErr := loader()
		if loadErr != nil {
			return nil, false, loadErr
		}

		payload, marshalErr := json.Marshal(loaded)
		if marshalErr != nil {
			return nil, false, marshalErr
		}

		if setErr := s.client.Set(ctx, key, payload, ttl).Err(); setErr != nil {
			return nil, false, setErr
		}

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

func (s *RedisStore) doInflightLoad(key string, loader func() ([]byte, bool, error)) ([]byte, bool, error) {
	s.inflightMu.Lock()
	if call, exists := s.inflight[key]; exists {
		s.inflightMu.Unlock()
		<-call.done
		return call.payload, call.hit, call.err
	}

	call := &redisInflightCall{done: make(chan struct{})}
	s.inflight[key] = call
	s.inflightMu.Unlock()

	call.payload, call.hit, call.err = loader()
	close(call.done)

	s.inflightMu.Lock()
	delete(s.inflight, key)
	s.inflightMu.Unlock()

	return call.payload, call.hit, call.err
}

func (s *RedisStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := s.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return err
	}

	RecordSet()
	return nil
}

func (s *RedisStore) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	RecordInvalidate()
	return nil
}

func (s *RedisStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, prefix+"*", 200).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := s.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			for range keys {
				RecordInvalidate()
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}
