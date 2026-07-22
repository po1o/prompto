package cache

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/maps"
)

type store struct {
	cache *maps.Concurrent[*Entry[any]]
	// dirty is written from concurrent daemon render goroutines (via Set/Delete
	// on the shared global stores), so it must be atomic. It is currently
	// write-only: persistence is stubbed out (see close), but the flag is kept
	// for when it returns.
	dirty atomic.Bool
}

var (
	// storeMu guards lazy creation of the session/device globals below. The
	// daemon serves renders from concurrent goroutines, so the nil-check-then-
	// assign in get() would otherwise race.
	storeMu sync.Mutex
	session *store
	device  *store
)

type Store string

const (
	Session Store  = "session"
	Device  Store  = "device"
	TTL     string = "ttl"
)

func (s Store) new() *store {
	return &store{
		cache: maps.NewConcurrent[*Entry[any]](),
	}
}

// getStore returns the appropriate store based on the Store identifier
func (s Store) get() *store {
	storeMu.Lock()
	defer storeMu.Unlock()

	switch s { //nolint:exhaustive
	case Device:
		if device == nil {
			device = s.new()
		}

		return device
	default:
		if session == nil {
			session = s.new()
		}

		return session
	}
}

// Init initializes a store.
func (s Store) init(filePath string) {
	defer log.Trace(time.Now(), string(s), filePath)

	store := s.get()
	// Clear in place rather than reassigning store.cache: a render goroutine may
	// be reading the field concurrently, and swapping the pointer would race.
	store.cache.Clear()
	store.dirty.Store(false)
}

func (s Store) close() {
	defer log.Trace(time.Now(), string(s))

	log.Debugf("(%s) not persisting", string(s))
}

// Get retrieves a typed value from the specified store
func Get[T any](s Store, key string) (T, bool) {
	var zero T
	defer log.Trace(time.Now(), string(s), key)

	store := s.get()
	if store == nil {
		log.Debugf("(%s) store is nil", string(s))
		return zero, false
	}

	entry, found := store.cache.Get(key)
	if !found {
		log.Debugf("(%s) key not found: %s", string(s), key)
		return zero, false
	}

	if entry.Expired() {
		log.Debugf("(%s) key expired: %s", string(s), key)
		store.cache.Delete(key)
		store.dirty.Store(true)
		return zero, false
	}

	// Type assertion to get the typed value
	if typed, ok := entry.Value.(T); ok {
		log.Debugf("(%s) found entry: %s - %v", string(s), key, typed)
		return typed, true
	}

	log.Error(fmt.Errorf("(%s) type mismatch for key: %s. Got %T, expected %T", string(s), key, entry.Value, zero))
	return zero, false
}

// Set stores a typed value in the specified store
func Set[T any](s Store, key string, value T, duration Duration) {
	defer log.Trace(time.Now(), string(s), key)

	store := s.get()
	if store == nil {
		log.Debugf("(%s) store is nil", string(s))
		return
	}

	seconds := duration.Seconds()
	if seconds == 0 {
		return
	}

	log.Debugf("(%s) setting entry: %s - %v with duration: %s", string(s), key, value, string(duration))

	store.cache.Set(key, &Entry[any]{
		Value:     value,
		Timestamp: time.Now().Unix(),
		TTL:       seconds,
	})

	store.dirty.Store(true)
}

// Delete removes a key from the specified store
func Delete(s Store, key string) {
	defer log.Trace(time.Now(), string(s), key)

	store := s.get()
	if store == nil {
		log.Debugf("(%s) store is nil", string(s))
		return
	}

	log.Debugf("(%s) deleting key: %s", string(s), key)
	store.cache.Delete(key)
	store.dirty.Store(true)
}

func DeleteAll(s Store) {
	defer log.Trace(time.Now(), string(s))

	store := s.get()
	if store == nil {
		log.Debugf("(%s) store is nil", string(s))
		return
	}

	// Clear in place rather than reassigning: CacheClear invokes DeleteAll from
	// its own gRPC handler goroutine, concurrent with render goroutines reading
	// store.cache, so swapping the pointer would race on the field.
	store.cache.Clear()
	store.dirty.Store(true)
}

func Print(s Store) string {
	defer log.Trace(time.Now(), string(s))

	store := s.get()
	if store == nil {
		return fmt.Sprintf("Store %s is nil", string(s))
	}

	cache := store.cache.ToSimple()
	if len(cache) == 0 {
		return fmt.Sprintf("Store %s is empty", string(s))
	}

	var builder strings.Builder

	for key, entry := range cache {
		builder.WriteString("\n")

		if entry.Expired() {
			fmt.Fprintf(&builder, "Key: %s [EXPIRED]\n", key)
			builder.WriteString("\n")
			continue
		}

		var ttlInfo string
		if entry.TTL < 0 {
			ttlInfo = "never expires"
		}
		if entry.TTL >= 0 {
			expiresAt := time.Unix(entry.Timestamp+int64(entry.TTL), 0)
			ttlInfo = fmt.Sprintf("expires at %s", expiresAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Fprintf(&builder, "Key: %s\n", key)
		fmt.Fprintf(&builder, "  Value: %#v\n", entry.Value)
		fmt.Fprintf(&builder, "  Type: %T\n", entry.Value)
		fmt.Fprintf(&builder, "  Created: %s\n", time.Unix(entry.Timestamp, 0).Format("2006-01-02 15:04:05"))
		fmt.Fprintf(&builder, "  TTL: %s\n", ttlInfo)
	}

	return builder.String()
}
