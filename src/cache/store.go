package cache

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/maps"
)

type store struct {
	cache *maps.Concurrent[*Entry[any]]
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

// init resets a store. Nothing is read from or written to disk: both stores
// live for the lifetime of the process.
func (s Store) init() {
	defer log.Trace(time.Now(), string(s))

	store := s.get()
	// Clear in place rather than reassigning store.cache: a render goroutine may
	// be reading the field concurrently, and swapping the pointer would race.
	store.cache.Clear()
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
}

// Item is a flattened, display-ready view of one cached value. The stored
// entry is not exposed: callers only ever render it.
type Item struct {
	CreatedAt time.Time
	ExpiresAt time.Time
	Key       string
	Value     string
	Type      string
	Expired   bool
	Forever   bool
}

// Items lists a store's contents, sorted by key so repeated calls read the
// same way. Expired entries are included rather than dropped: seeing that a
// value went stale is usually the reason for looking.
func Items(s Store) []Item {
	defer log.Trace(time.Now(), string(s))

	store := s.get()
	if store == nil {
		return nil
	}

	entries := store.cache.ToSimple()
	items := make([]Item, 0, len(entries))

	for key, entry := range entries {
		if entry == nil {
			continue
		}

		item := Item{
			Key:       key,
			Value:     fmt.Sprintf("%v", entry.Value),
			Type:      fmt.Sprintf("%T", entry.Value),
			CreatedAt: time.Unix(entry.Timestamp, 0),
			Expired:   entry.Expired(),
			Forever:   entry.TTL < 0,
		}

		if !item.Forever {
			item.ExpiresAt = time.Unix(entry.Timestamp+int64(entry.TTL), 0)
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b Item) int {
		return strings.Compare(a.Key, b.Key)
	})

	return items
}
