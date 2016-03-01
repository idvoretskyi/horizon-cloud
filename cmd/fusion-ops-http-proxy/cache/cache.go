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
	mu   sync.Mutex
	m    map[string]*cacheEntry
	fill func(string) (string, error)
}

type cacheEntry struct {
	value   string
	expires time.Time
}

func New(fill func(string) (string, error)) *Cache {
	return &Cache{
		m:    make(map[string]*cacheEntry, 128),
		fill: fill,
	}
}

func (c *Cache) Get(key string) (string, error) {
	start := time.Now()

	c.mu.Lock()
	ent, ok := c.m[key]
	if ok && ent.expires.After(start) {
		delete(c.m, key)
		ok = false
	}
	c.mu.Unlock()
	if ok {
		return ent.value, nil
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
