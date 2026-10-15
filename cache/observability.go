package cache

import (
	"context"
	"sync/atomic"
)

type Metrics struct {
	Backend    string `json:"backend"`
	Hit        uint64 `json:"hit"`
	Miss       uint64 `json:"miss"`
	Set        uint64 `json:"set"`
	Invalidate uint64 `json:"invalidate"`
}

var cacheMetrics struct {
	backend atomic.Value
	hit     atomic.Uint64
	miss    atomic.Uint64
	set     atomic.Uint64
	inv     atomic.Uint64
}

func SetBackend(backend string) {
	cacheMetrics.backend.Store(backend)
}

func RecordHit() {
	cacheMetrics.hit.Add(1)
}

func RecordMiss() {
	cacheMetrics.miss.Add(1)
}

func RecordSet() {
	cacheMetrics.set.Add(1)
}

func RecordInvalidate() {
	cacheMetrics.inv.Add(1)
}

func GetMetrics() Metrics {
	backend := ""
	if value := cacheMetrics.backend.Load(); value != nil {
		if typed, ok := value.(string); ok {
			backend = typed
		}
	}

	return Metrics{
		Backend:    backend,
		Hit:        cacheMetrics.hit.Load(),
		Miss:       cacheMetrics.miss.Load(),
		Set:        cacheMetrics.set.Load(),
		Invalidate: cacheMetrics.inv.Load(),
	}
}

type traceContextKey struct{}

type RequestTrace struct {
	Backend string
	Key     string
	Hit     bool
	HasHit  bool
}

func WithRequestTrace(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Value(traceContextKey{}).(*RequestTrace); ok {
		return ctx
	}

	return context.WithValue(ctx, traceContextKey{}, &RequestTrace{})
}

func GetRequestTrace(ctx context.Context) (RequestTrace, bool) {
	if ctx == nil {
		return RequestTrace{}, false
	}

	trace, ok := ctx.Value(traceContextKey{}).(*RequestTrace)
	if !ok || trace == nil {
		return RequestTrace{}, false
	}

	return *trace, true
}

func RecordTrace(ctx context.Context, backend, key string, hit bool, hasHit bool) {
	if ctx == nil {
		return
	}

	trace, ok := ctx.Value(traceContextKey{}).(*RequestTrace)
	if !ok || trace == nil {
		return
	}

	trace.Backend = backend
	trace.Key = key
	trace.Hit = hit
	trace.HasHit = hasHit
}
