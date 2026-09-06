package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreItems(t *testing.T) {
	cases := []struct {
		setupFunc func() *store
		testFunc  func(t *testing.T)
		name      string
	}{
		{
			name: "store with data",
			setupFunc: func() *store {
				testStore := Session.new()
				testStore.cache.Set("test_key1", &Entry[any]{
					Value:     "test_value1",
					Timestamp: time.Now().Unix(),
					TTL:       3600, // 1 hour
				})
				testStore.cache.Set("test_key2", &Entry[any]{
					Value:     42,
					Timestamp: time.Now().Unix(),
					TTL:       -1, // never expires
				})
				testStore.cache.Set("expired_key", &Entry[any]{
					Value:     "expired_value",
					Timestamp: time.Now().Unix() - 7200, // 2 hours ago
					TTL:       3600,                     // 1 hour (should be expired)
				})
				session = testStore
				return testStore
			},
			testFunc: func(t *testing.T) {
				items := Items(Session)
				require.Len(t, items, 3)

				// Sorted by key, so repeated calls read the same way.
				assert.Equal(t, "expired_key", items[0].Key)
				assert.Equal(t, "test_key1", items[1].Key)
				assert.Equal(t, "test_key2", items[2].Key)

				// An expired entry is reported rather than dropped: seeing that
				// a value went stale is usually the reason for looking.
				assert.True(t, items[0].Expired)

				assert.Equal(t, "test_value1", items[1].Value)
				assert.Equal(t, "string", items[1].Type)
				assert.False(t, items[1].Forever)
				assert.False(t, items[1].ExpiresAt.IsZero())

				assert.Equal(t, "42", items[2].Value)
				assert.Equal(t, "int", items[2].Type)
				assert.True(t, items[2].Forever)
				assert.True(t, items[2].ExpiresAt.IsZero(), "an entry that never expires has no expiry")
			},
		},
		{
			name: "empty store",
			setupFunc: func() *store {
				testStore := Session.new()
				session = testStore
				return testStore
			},
			testFunc: func(t *testing.T) {
				assert.Empty(t, Items(Session))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFunc()
			tc.testFunc(t)
		})
	}
}

// TestStoreConcurrentAccessWithClear drives Set/Get/Delete concurrently with
// DeleteAll to guard against the store.cache pointer race that DeleteAll used to
// introduce (it reassigned the field while readers held the old pointer). Run
// with -race to catch a regression.
func TestStoreConcurrentAccessWithClear(t *testing.T) {
	session = Session.new()

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers + 1)

	stop := make(chan struct{})

	for i := range workers {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			for {
				select {
				case <-stop:
					return
				default:
					Set(Session, key, id, ONEDAY)
					_, _ = Get[int](Session, key)
					Delete(Session, key)
				}
			}
		}(i)
	}

	go func() {
		defer wg.Done()
		for range 500 {
			DeleteAll(Session)
		}
		close(stop)
	}()

	wg.Wait()
}

// Items reads the store while other goroutines write it, so it must not hand
// back a view that can be mutated underneath the caller.
func TestItemsConcurrentWithWrites(t *testing.T) {
	session = Session.new()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				Set(Session, fmt.Sprintf("key-%d", i%16), i, ONEDAY)
			}
		}
	})

	wg.Go(func() {
		defer close(stop)
		for range 500 {
			_ = Items(Session)
		}
	})

	wg.Wait()
}
