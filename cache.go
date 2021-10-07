package aliyundrive

import (
	"bytes"
	"encoding/gob"
	"github.com/allegro/bigcache/v3"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type bigCache struct {
	cache *bigcache.BigCache
}

type bigCacheOptions struct {
	size      int
	ttl       time.Duration
	cleanFreq time.Duration
}

func newBigCache(options *bigCacheOptions) (*bigCache, error) {
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})

	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             16,
		LifeWindow:         options.ttl,
		CleanWindow:        options.cleanFreq,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		Verbose:            false,
		HardMaxCacheSize:   options.size,
		StatsEnabled:       true,
	})

	if err != nil {
		return nil, err
	}

	return &bigCache{
		cache: cache,
	}, nil
}

func (b *bigCache) Get(key string) (interface{}, error) {
	value, err := b.cache.Get(key)
	if err != nil {
		return nil, err
	}

	v, err := deserialize(value)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (b *bigCache) Set(key string, value interface{}) error {
	valueBytes, err := serialize(value)
	if err != nil {
		logrus.Errorf("serialize error %s", err)
		return err
	}

	err = b.cache.Set(key, valueBytes)
	if err != nil {
		return err
	}

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

func serialize(value interface{}) ([]byte, error) {
	buffer := bytes.Buffer{}
	enc := gob.NewEncoder(&buffer)
	gob.Register(value)

	err := enc.Encode(&value)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func deserialize(valueBytes []byte) (interface{}, error) {
	var value interface{}
	buf := bytes.NewBuffer(valueBytes)
	dec := gob.NewDecoder(buf)

	err := dec.Decode(&value)
	if err != nil {
		return nil, err
	}

	return value, nil
}
