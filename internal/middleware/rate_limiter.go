package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type client struct {
	mu         sync.Mutex
	tokens     float64
	lastUpdate time.Time
}

type RateLimiter struct {
	sync.RWMutex
	clients  map[string]*client
	rate     float64       // сколько токенов добавляется в секунду
	capacity float64       // максимальный баланс бакета
	ttl      time.Duration // время жизни неактивного клиента в памяти
}

func NewRateLimiter(maxRequests int, perWindow time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*client),
		rate:     float64(maxRequests) / perWindow.Seconds(),
		capacity: float64(maxRequests),
		ttl:      time.Hour, // удалять неактивных клиентов через 1 час
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		c := rl.getClient(ip)

		c.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(c.lastUpdate).Seconds()
		c.lastUpdate = now

		c.tokens += elapsed * rl.rate
		if c.tokens > rl.capacity {
			c.tokens = rl.capacity
		}

		if c.tokens < 1.0 {
			c.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Too many requests. Please try again later."}`))
			return
		}

		c.tokens -= 1.0
		c.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getClient(ip string) *client {
	rl.RLock()
	c, exists := rl.clients[ip]
	rl.RUnlock()

	if exists {
		return c
	}

	rl.Lock()
	if c, exists = rl.clients[ip]; !exists {
		c = &client{
			tokens:     rl.capacity,
			lastUpdate: time.Now(),
		}
		rl.clients[ip] = c
	}
	rl.Unlock()

	return c
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.Lock()
		now := time.Now()
		for ip, c := range rl.clients {
			c.mu.Lock()
			if now.Sub(c.lastUpdate) > rl.ttl {
				delete(rl.clients, ip)
			}
			c.mu.Unlock()
		}
		rl.Unlock()
	}
}
