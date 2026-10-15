package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString generates a cryptographically secure random string of the specified length.
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails (very unlikely)
		return fmt.Sprintf("%d", time.Now().UnixNano())[:length]
	}

	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}
	return string(bytes)
}

// GenerateRandomHex generates a random hex string of the specified byte length
func GenerateRandomHex(byteLength int) string {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// GenerateSecureToken generates a secure token for things like password reset
func GenerateSecureToken() string {
	return GenerateRandomHex(32) // 64 character hex string
}

// SetOAuthCookie sets a secure OAuth cookie
func SetOAuthCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,                  // MUST be true on HTTPS
		SameSite: http.SameSiteNoneMode, // MUST be None for cross-site
	})
}

// SetCookie sets a standard secure cookie
func SetCookie(c *gin.Context, name, value string, maxAge int, secure bool) {
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

// ClearOAuthCookie clears an OAuth cookie
func ClearOAuthCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
}

// ClearCookie clears a cookie
func ClearCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

// OAuthState holds OAuth state information
type OAuthState struct {
	State       string
	RedirectURI string
	IsMobile    bool
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// OAuthStateStore manages OAuth state tokens
type OAuthStateStore struct {
	states map[string]*OAuthState
	mu     sync.RWMutex
}

var globalStateStore *OAuthStateStore
var stateStoreOnce sync.Once

// GetOAuthStateStore returns the global state store (singleton)
func GetOAuthStateStore() *OAuthStateStore {
	stateStoreOnce.Do(func() {
		globalStateStore = &OAuthStateStore{
			states: make(map[string]*OAuthState),
		}
		// Start cleanup goroutine
		go globalStateStore.cleanupExpired()
	})
	return globalStateStore
}

// Set stores an OAuth state with expiration
func (s *OAuthStateStore) Set(state string, redirectURI string, isMobile bool, ttlSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.states[state] = &OAuthState{
		State:       state,
		RedirectURI: redirectURI,
		IsMobile:    isMobile,
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Duration(ttlSeconds) * time.Second),
	}
}

// Get retrieves and REMOVES an OAuth state (one-time use)
func (s *OAuthStateStore) Get(state string) (*OAuthState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oauthState, exists := s.states[state]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(oauthState.ExpiresAt) {
		delete(s.states, state)
		return nil, false
	}

	// Remove state (one-time use for security)
	delete(s.states, state)

	return oauthState, true
}

// Peek retrieves an OAuth state without removing it
func (s *OAuthStateStore) Peek(state string) (*OAuthState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	oauthState, exists := s.states[state]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(oauthState.ExpiresAt) {
		return nil, false
	}

	return oauthState, true
}

// Delete removes an OAuth state
func (s *OAuthStateStore) Delete(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, state)
}

// cleanupExpired removes expired states every minute
func (s *OAuthStateStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for state, oauthState := range s.states {
			if now.After(oauthState.ExpiresAt) {
				delete(s.states, state)
			}
		}
		s.mu.Unlock()
	}
}

// Count returns the number of active states (for monitoring)
func (s *OAuthStateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to a time.Time
func TimePtr(t time.Time) *time.Time {
	return &t
}

// Coalesce returns the first non-empty string
func Coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Contains checks if a slice contains a value
func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Unique returns a slice with unique values
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
