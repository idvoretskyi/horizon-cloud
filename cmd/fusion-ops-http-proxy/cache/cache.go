package cache

import (
	"errors"
	"sync"
	"time"
)

var (
	cacheTimeout = time.Second * 15
	ErrNotFound  = errors.New("not found")
)

// TODO: implement meetup semantics for backpressure
// TODO: implement stale data usage and background revalidation

type Cache struct {
	fill func(string) (string, error)

	mu sync.Mutex
	m  map[string]*cacheEntry
}

type cacheEntry struct {
	value   string
	expires time.Time
}

func New(fill func(string) (string, error)) *Cache {
	return &Cache{
		fill: fill,
		m:    make(map[string]*cacheEntry, 128),
	}
}

func (c *Cache) Get(key string) (string, error) {
	start := time.Now()

	c.mu.Lock()
	entry, ok := c.m[key]
	if ok && entry.expires.Before(start) {
		delete(c.m, key)
		ok = false
	}
	c.mu.Unlock()
	if ok {
		return entry.value, nil
	}

	value, err := c.fill(key)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.m[key] = &cacheEntry{
		value:   value,
		expires: start.Add(cacheTimeout),
	}
	c.mu.Unlock()

	return value, nil
}
