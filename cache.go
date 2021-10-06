package aliyundrive

import (
	"errors"
	"github.com/allegro/bigcache/v3"
	"strings"
	"sync"
	"time"
)

type bigCache struct {
	cache    *bigcache.BigCache
	cacheMap *sync.Map
}

type bigCacheOptions struct {
	size      int
	ttl       time.Duration
	cleanFreq time.Duration
}

var ErrEntryNotFound = errors.New("entry not found")

func newBigCache(options *bigCacheOptions) (*bigCache, error) {
	m := &sync.Map{}

	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             16,
		LifeWindow:         options.ttl,
		CleanWindow:        options.cleanFreq,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		Verbose:            false,
		HardMaxCacheSize:   options.size,
		StatsEnabled:       true,
		OnRemove: func(key string, entry []byte) {
			m.Delete(key)
		},
	})

	if err != nil {
		return nil, err
	}

	return &bigCache{
		cache:    cache,
		cacheMap: m,
	}, nil
}

func (b *bigCache) Get(key string) (interface{}, error) {
	if value, ok := b.cacheMap.Load(key); ok {
		return value, nil
	}

	return nil, ErrEntryNotFound
}

func (b *bigCache) Set(key string, value interface{}) error {
	b.cacheMap.Store(key, value)
	_ = b.cache.Set(key, []byte("empty"))

	return nil
}

func (b *bigCache) RemoveWithPrefix(prefix string) int {
	iterator := b.cache.Iterator()
	count := 0

	for iterator.SetNext() {
		value, err := iterator.Value()
		if err != nil {
			return count
		}

		if strings.HasPrefix(value.Key(), prefix) {
			_ = b.cache.Delete(value.Key())
			count++
		}
	}

	return count
}
