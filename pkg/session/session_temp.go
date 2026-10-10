// Package session manages WebSocket connections and player sessions.
package session

import "github.com/zax0rz/darkpawns/pkg/dprng"

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

// RandomInt generates a random integer in range [0, n) from the canonical stream.
func (s *Session) RandomInt(n int) int {
	if n <= 0 {
		return 0
	}
	return dprng.Number(0, n-1)
}

// maybeRefreshToken checks if the session's JWT is within the refresh
// window (15 minutes before the 1-hour effective expiry). If so, it generates
// a new token and sends it to the client as a token_refresh message.
