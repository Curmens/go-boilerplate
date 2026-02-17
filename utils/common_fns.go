package utils

import (
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString generates a random string of the specified length.
func GenerateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func SetOAuthCookie(c *gin.Context, name, value string, maxAge int) {
	// devtunnels is HTTPS -> must use Secure + SameSite=None for cross-site redirect
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,                  // MUST be true on HTTPS
		SameSite: http.SameSiteNoneMode, // MUST be None for cross-site
		// Domain: optional. Usually omit to default to current host.
	})
}

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

func init() {
	globalStateStore = &OAuthStateStore{
		states: make(map[string]*OAuthState),
	}

	// Start cleanup goroutine
	go globalStateStore.cleanupExpired()
}

// GetOAuthStateStore returns the global state store
func GetOAuthStateStore() *OAuthStateStore {
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
