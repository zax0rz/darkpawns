package optimization

import "errors"

var (
	// Pool errors
	ErrPoolClosed    = errors.New("pool is closed")
	ErrPoolFull      = errors.New("pool is full")
	ErrPoolExhausted = errors.New("connection pool exhausted")

	// WebSocket errors
	ErrWebSocketBufferFull = errors.New("WebSocket buffer full")
	ErrWebSocketTimeout    = errors.New("WebSocket operation timeout")

	// Database errors
	ErrDBConnectionFailed = errors.New("database connection failed")
	ErrDBQueryTimeout     = errors.New("database query timeout")

	// Cache errors
	ErrCacheMiss    = errors.New("cache miss")
	ErrCacheInvalid = errors.New("cache invalid")

	// Rate limiting errors
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
