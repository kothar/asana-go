package asana

import (
	"time"
)

// Cache records the results of API requests and replays them if requested
// again while the value is not expired
type Cache interface {
	Put(key string, value []byte) error
	Get(key string) ([]byte, error)
	Clear(key string) error
}

type cacheEntry struct {
	expires time.Time
	value   []byte
}

// MapCache implements the Cache interface using a standard Go map
type MapCache struct {
	expiry  time.Duration
	entries map[string]*cacheEntry
}

// NewMapCache creates a new MapCache instance with the given expiry duration
func NewMapCache(expiry time.Duration) (*MapCache, error) {
	return &MapCache{
		expiry:  expiry,
		entries: make(map[string]*cacheEntry),
	}, nil
}

// Put stores a value in the cache, replacing any existing entry and updating the expiry time
func (c *MapCache) Put(key string, value []byte) error {
	c.entries[key] = &cacheEntry{
		expires: time.Now().Add(c.expiry),
		value:   value,
	}
	return nil
}

// Get retrieves a previously stored value from the cache, if present
func (c *MapCache) Get(key string) ([]byte, error) {
	entry := c.entries[key]
	if entry == nil {
		return nil, nil
	} else if entry.expires.Before(time.Now()) {
		c.Clear(key)
		return nil, nil
	}

	return entry.value, nil
}

// Clear removes a previously stored value from the cache, if present
func (c *MapCache) Clear(key string) error {
	delete(c.entries, key)
	return nil
}

func (c *Client) cache(path string, body []byte) {
	if c.Cache == nil {
		return
	}

	c.debug("Caching response for %s", path)
	c.Cache.Put(path, body)
}

func (c *Client) getCached(path string) []byte {
	if c.Cache == nil {
		return nil
	}

	c.debug("Check for cached response for %s", path)
	value, err := c.Cache.Get(path)
	if err != nil || value == nil {
		return nil
	}

	c.info("Using cached response for %s", path)
	return value
}

func (c *Client) clearCache(path string) {
	if c.Cache == nil {
		return
	}

	c.debug("Clearing cache for %s", path)
	c.Cache.Clear(path)
}
