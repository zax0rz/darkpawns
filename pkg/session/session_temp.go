// Package session manages WebSocket connections and player sessions.
package session

import "math/rand"

func (s *Session) SetTempData(key string, value interface{}) {
	if s.tempData == nil {
		s.tempData = make(map[string]interface{})
	}
	s.tempData[key] = value
}

// GetTempData retrieves temporary data from the session
func (s *Session) GetTempData(key string) interface{} {
	if s.tempData == nil {
		return nil
	}
	return s.tempData[key]
}

// ClearTempData removes temporary data from the session
func (s *Session) ClearTempData(key string) {
	if s.tempData != nil {
		delete(s.tempData, key)
	}
}

// GetTempInt retrieves temporary data as an int.
// Returns the value and true if the key exists and is an int, zero and false otherwise.
func (s *Session) GetTempInt(key string) (int, bool) {
	if s.tempData == nil {
		return 0, false
	}
	v, ok := s.tempData[key]
	if !ok {
		return 0, false
	}
	i, ok := v.(int)
	return i, ok
}

// GetTempString retrieves temporary data as a string.
// Returns the value and true if the key exists and is a string, empty and false otherwise.
func (s *Session) GetTempString(key string) (string, bool) {
	if s.tempData == nil {
		return "", false
	}
	v, ok := s.tempData[key]
	if !ok {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}

// SetTemp stores temporary data in the session.
func (s *Session) SetTemp(key string, value interface{}) {
	if s.tempData == nil {
		s.tempData = make(map[string]interface{})
	}
	s.tempData[key] = value
}

// RandomInt generates a random integer in range [0, n)
func (s *Session) RandomInt(n int) int {
	if n <= 0 {
		return 0
	}
	// Use math/rand for randomness
	// Note: In production, you might want to use a cryptographically secure random source
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	return rand.Intn(n)
}

// maybeRefreshToken checks if the session's JWT is within the refresh
// window (15 minutes before the 1-hour effective expiry). If so, it generates
// a new token and sends it to the client as a token_refresh message.
