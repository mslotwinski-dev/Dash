package app

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// CacheItem przechowuje zbuforowaną odpowiedź z backendu
type CacheItem struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

// Cache to bezpieczna wielowątkowo mapa w pamięci RAM
type Cache struct {
	sync.RWMutex
	items map[string]CacheItem
	ttl   time.Duration // Czas życia wpisu, np. 10 sekund
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
		ttl:   ttl,
	}
}

func (c *Cache) Get(key string) (CacheItem, bool) {
	c.RLock()
	defer c.RUnlock()
	item, found := c.items[key]
	if !found {
		return CacheItem{}, false
	}
	// Sprawdzamy czy wpis nie wygasł
	if time.Now().After(item.ExpiresAt) {
		return CacheItem{}, false
	}
	return item, true
}

func (c *Cache) Set(key string, statusCode int, header http.Header, body []byte) {
	c.Lock()
	defer c.Unlock()
	c.items[key] = CacheItem{
		StatusCode: statusCode,
		Header:     header,
		Body:       body,
		ExpiresAt:  time.Now().Add(c.ttl),
	}
}

// CacheResponseWriter służy do "podglądania" i zapisywania tego, co backend wysyła do użytkownika
type cacheResponseWriter struct {
	http.ResponseWriter
	bodyBuf    *bytes.Buffer
	statusCode int
}

func (crw *cacheResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

func (crw *cacheResponseWriter) Write(b []byte) (int, error) {
	crw.bodyBuf.Write(b) // Zapisujemy kopię do naszego bufora w pamięci RAM
	return crw.ResponseWriter.Write(b)
}
