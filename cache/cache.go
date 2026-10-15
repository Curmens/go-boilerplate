package cache

import (
	"context"
	"time"

	"example.com/config"
	"example.com/utils"
)

// Store defines cache operations used by services.
type Store interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, loader func() (interface{}, error)) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
	Close() error
	Backend() string
}

// NewStore creates either a Redis or local-memory cache based on configuration.
func NewStore(cfg *config.Config, logger *utils.Logger) (Store, error) {
	if cfg.Cache.UseRedis {
		redisStore, err := NewRedisStore(cfg.Redis)
		if err != nil {
			return nil, err
		}
		SetBackend(redisStore.Backend())
		if logger != nil {
			_ = logger.Info("cache backend selected", map[string]interface{}{"backend": redisStore.Backend()})
		}
		return redisStore, nil
	}

	localStore := NewLocalStore()
	SetBackend(localStore.Backend())
	if logger != nil {
		_ = logger.Info("cache backend selected", map[string]interface{}{"backend": localStore.Backend()})
	}
	return localStore, nil
}
