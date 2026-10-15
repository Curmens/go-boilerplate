package utils

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestManager(t *testing.T, ttl time.Duration) *JWTManager {
	t.Helper()
	m, err := NewJWTManager(JWTConfig{Secret: strings.Repeat("s", 32), AccessTokenTTL: ttl})
	if err != nil {
		t.Fatalf("NewJWTManager: %v", err)
	}
	return m
}

func TestNewJWTManagerRejectsShortSecret(t *testing.T) {
	if _, err := NewJWTManager(JWTConfig{Secret: "short"}); err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	m := newTestManager(t, time.Minute)

	pair, err := m.GenerateTokenPair(42, "a@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	claims, err := m.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.UserID != 42 || claims.Email != "a@example.com" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := m.ValidateRefreshToken(pair.RefreshToken); err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
}

func TestRefreshTokenIsNotAnAccessToken(t *testing.T) {
	m := newTestManager(t, time.Minute)

	refresh, err := m.GenerateRefreshToken(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ValidateAccessToken(refresh); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("refresh token accepted as access token (err=%v)", err)
	}

	access, _, err := m.GenerateAccessToken(1, "a@example.com", "user")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ValidateRefreshToken(access); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
}

func TestExpiredAccessToken(t *testing.T) {
	m := newTestManager(t, -time.Minute)

	token, _, err := m.GenerateAccessToken(1, "a@example.com", "user")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.ValidateAccessToken(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestTokenSignedWithOtherSecret(t *testing.T) {
	m := newTestManager(t, time.Minute)
	other, _ := NewJWTManager(JWTConfig{Secret: strings.Repeat("x", 32)})

	token, _, _ := other.GenerateAccessToken(1, "a@example.com", "user")
	if _, err := m.ValidateAccessToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestNewMeta(t *testing.T) {
	cases := []struct {
		page, perPage int
		total         int64
		wantPages     int
		wantNext      bool
	}{
		{1, 10, 0, 0, false},
		{1, 10, 10, 1, false},
		{1, 10, 11, 2, true},
		{2, 10, 11, 2, false},
	}
	for _, tc := range cases {
		m := NewMeta(tc.page, tc.perPage, tc.total)
		if m.TotalPages != tc.wantPages || m.HasNext != tc.wantNext {
			t.Errorf("NewMeta(%d,%d,%d) = pages %d next %v; want %d %v",
				tc.page, tc.perPage, tc.total, m.TotalPages, m.HasNext, tc.wantPages, tc.wantNext)
		}
	}
}
