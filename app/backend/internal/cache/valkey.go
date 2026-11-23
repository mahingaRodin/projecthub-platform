package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valkey-io/valkey-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	keyPrefix     = "projecthub:"
	opTimeout     = 300 * time.Millisecond
	retryInterval = 5 * time.Second
	warnInterval  = 30 * time.Second
)

// Valkey is a Cache backed by a Valkey (or Redis-compatible) server.
// If the server is unreachable at startup the API still starts; a background loop keeps trying to connect.
type Valkey struct {
	opt      valkey.ClientOption
	mu       sync.RWMutex
	client   valkey.Client
	lastWarn atomic.Int64
	stop     chan struct{}
}

func NewValkey(addr, password string, db int) *Valkey {
	v := &Valkey{
		opt: valkey.ClientOption{
			InitAddress:      []string{addr},
			Password:         password,
			SelectDB:         db,
			DisableCache:     true, // no client-side caching: keep behaviour simple and predictable
			DisableRetry:     true, // fail immediately during an outage instead of retrying until the timeout
			Dialer:           net.Dialer{Timeout: 500 * time.Millisecond},
			ConnWriteTimeout: 500 * time.Millisecond,
		},
		stop: make(chan struct{}),
	}
	if !v.connect() {
		go v.reconnectLoop()
	}
	return v
}

func (v *Valkey) connect() bool {
	c, err := valkey.NewClient(v.opt)
	if err != nil {
		v.warn("cache unavailable, using MongoDB only: %v", err)
		return false
	}
	v.mu.Lock()
	v.client = c
	v.mu.Unlock()
	log.Printf("connected to Valkey cache at %s", v.opt.InitAddress[0])
	return true
}

func (v *Valkey) reconnectLoop() {
	t := time.NewTicker(retryInterval)
	defer t.Stop()
	for {
		select {
		case <-v.stop:
			return
		case <-t.C:
			if v.connect() {
				return
			}
		}
	}
}

func (v *Valkey) get() valkey.Client {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.client
}

// warn logs at most once per warnInterval so an outage does not flood the logs.
func (v *Valkey) warn(format string, args ...any) {
	now := time.Now().UnixNano()
	last := v.lastWarn.Load()
	if now-last < int64(warnInterval) || !v.lastWarn.CompareAndSwap(last, now) {
		return
	}
	log.Printf("WARN "+format, args...)
}

func (v *Valkey) Get(ctx context.Context, key string, dest any) bool {
	c := v.get()
	if c == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	raw, err := c.Do(ctx, c.B().Get().Key(keyPrefix+key).Build()).AsBytes()
	if valkey.IsValkeyNil(err) {
		return false
	}
	if err != nil {
		v.warn("cache get %q failed: %v", key, err)
		return false
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		v.warn("cache value %q is not valid JSON: %v", key, err)
		return false
	}
	return true
}

func (v *Valkey) Set(ctx context.Context, key string, value any, ttl time.Duration) {
	c := v.get()
	if c == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		v.warn("cache encode %q failed: %v", key, err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	if err := c.Do(ctx, c.B().Set().Key(keyPrefix+key).Value(valkey.BinaryString(raw)).Ex(ttl).Build()).Error(); err != nil {
		v.warn("cache set %q failed: %v", key, err)
	}
}

func (v *Valkey) Delete(ctx context.Context, keys ...string) {
	c := v.get()
	if c == nil || len(keys) == 0 {
		return
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = keyPrefix + k
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	if err := c.Do(ctx, c.B().Del().Key(full...).Build()).Error(); err != nil {
		v.warn("cache delete %v failed: %v", keys, err)
	}
}

func (v *Valkey) Version(ctx context.Context, group string) int64 {
	c := v.get()
	if c == nil {
		return -1
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	n, err := c.Do(ctx, c.B().Get().Key(keyPrefix+"version:"+group).Build()).AsInt64()
	if valkey.IsValkeyNil(err) {
		return 0
	}
	if err != nil {
		v.warn("cache version %q failed: %v", group, err)
		return -1
	}
	return n
}

func (v *Valkey) Bump(ctx context.Context, group string) {
	c := v.get()
	if c == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	if err := c.Do(ctx, c.B().Incr().Key(keyPrefix+"version:"+group).Build()).Error(); err != nil {
		v.warn("cache bump %q failed: %v", group, err)
	}
}

func (v *Valkey) Status(ctx context.Context) string {
	c := v.get()
	if c == nil {
		return "down"
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	if err := c.Do(ctx, c.B().Ping().Build()).Error(); err != nil {
		return "down"
	}
	return "up"
}

func (v *Valkey) Close() {
	close(v.stop)
	if c := v.get(); c != nil {
		c.Close()
	}
}

// Key helpers keep key names in one place.

func UserKey(id primitive.ObjectID) string { return "user:" + id.Hex() }

func StatsKey(version int64, userID primitive.ObjectID) string {
	return fmt.Sprintf("stats:v%d:%s", version, userID.Hex())
}

const StatsGroup = "stats"
