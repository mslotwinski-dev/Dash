package services

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type CacheItem struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

type Cache struct {
	sync.RWMutex
	items map[string]CacheItem
	ttl   time.Duration
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

type CacheResponseWriter struct {
	http.ResponseWriter
	bodyBuf    *bytes.Buffer
	statusCode int
}

func (crw *CacheResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

func (crw *CacheResponseWriter) Write(b []byte) (int, error) {
	crw.bodyBuf.Write(b)
	return crw.ResponseWriter.Write(b)
}

func NewCRW(w http.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		bodyBuf:        bytes.NewBuffer(nil),
		statusCode:     http.StatusOK,
	}
}

func (crw *CacheResponseWriter) GetStatusCode() int {
	return crw.statusCode
}

func (crw *CacheResponseWriter) GetHeader() http.Header {
	return crw.Header()
}

func (crw *CacheResponseWriter) GetBodyBuf() []byte {
	return crw.bodyBuf.Bytes()
}
