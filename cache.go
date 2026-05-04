package main

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	resp    *Response
	expires time.Time
}

var cache = struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}{entries: make(map[string]cacheEntry)}

func cacheGet(key string) (*Response, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	e, ok := cache.entries[key]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.resp, true
}

func cacheSet(key string, resp *Response, ttl time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.entries[key] = cacheEntry{resp: resp, expires: time.Now().Add(ttl)}
}

// ttlFromHeaders derives cache TTL from Cache-Control / Expires headers.
// Returns 0 if response must not be cached.
func ttlFromHeaders(headers map[string]string) time.Duration {
	cc := headers["cache-control"]
	if cc != "" {
		for _, dir := range strings.Split(cc, ",") {
			dir = strings.TrimSpace(dir)
			if dir == "no-store" || dir == "no-cache" {
				return 0
			}
			if strings.HasPrefix(dir, "max-age=") {
				secs, err := strconv.Atoi(strings.TrimPrefix(dir, "max-age="))
				if err == nil && secs > 0 {
					return time.Duration(secs) * time.Second
				}
			}
		}
	}
	if exp := headers["expires"]; exp != "" {
		t, err := time.Parse(time.RFC1123, exp)
		if err == nil && t.After(time.Now()) {
			return time.Until(t)
		}
	}
	return 60 * time.Second // default TTL for responses with no cache headers
}
