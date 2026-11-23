// Package cache provides an optional cache (Valkey) for the service layer.
// The cache is never required: every method fails open, so callers fall back to MongoDB.
package cache

import (
	"context"
	"time"
)

type Cache interface {
	// Get decodes the cached JSON value into dest. It reports false on a miss or if the cache is unavailable.
	Get(ctx context.Context, key string, dest any) bool
	// Set stores value as JSON for ttl. Errors are logged, never returned.
	Set(ctx context.Context, key string, value any, ttl time.Duration)
	// Delete removes keys. Errors are logged, never returned.
	Delete(ctx context.Context, keys ...string)
	// Version returns the current version of a key group (0 if never bumped), or -1 if the cache is
	// unavailable, in which case the caller must not read or write that group.
	Version(ctx context.Context, group string) int64
	// Bump increments a group's version, which invalidates every key built from the previous version.
	Bump(ctx context.Context, group string)
	// Status reports "up", "down" or "disabled" for the health endpoint.
	Status(ctx context.Context) string
	Close()
}

// Noop is used when caching is disabled (CACHE_ENABLED=false).
type Noop struct{}

func (Noop) Get(context.Context, string, any) bool           { return false }
func (Noop) Set(context.Context, string, any, time.Duration) {}
func (Noop) Delete(context.Context, ...string)               {}
func (Noop) Version(context.Context, string) int64           { return -1 }
func (Noop) Bump(context.Context, string)                    {}
func (Noop) Status(context.Context) string                   { return "disabled" }
func (Noop) Close()                                          {}
