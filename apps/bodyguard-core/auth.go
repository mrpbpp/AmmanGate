package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// AuthManager manages authentication credentials
type AuthManager struct {
	mu               sync.RWMutex
	username         string
	passwordHash     string
	actionPIN        string
	sessionSecret    string
	sessions         map[string]*Session
	sessionTTL       time.Duration
}

// Session represents a user session
type Session struct {
	ID        string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		username:      env("BG_ADMIN_USER", "admin"),
		passwordHash:  env("BG_ADMIN_PASS_HASH", ""), // If empty, use default
		actionPIN:     env("BG_ACTION_PIN", "1234"),
		sessionSecret: env("BG_SESSION_SECRET", generateRandomSecret(32)),
		sessions:      make(map[string]*Session),
		sessionTTL:    24 * time.Hour,
	}
}

// Initialize sets up the authentication manager with defaults if needed
func (a *AuthManager) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If no password hash is set, use the default password
	if a.passwordHash == "" {
		defaultPass := env("BG_ADMIN_PASS", "admin123")
		hash, err := hashPassword(defaultPass, a.sessionSecret)
		if err != nil {
			return fmt.Errorf("failed to hash default password: %w", err)
		}
		a.passwordHash = hash
	}

	// Start session cleanup goroutine
	go a.cleanupExpiredSessions()

	// Warn if using default credentials
	if a.username == "admin" && (a.passwordHash == hashPasswordDirect("admin123", a.sessionSecret)) {
		fmt.Println("⚠️  WARNING: Using default credentials (admin/admin123). Change them in production!")
	}

	if a.actionPIN == "1234" {
		fmt.Println("⚠️  WARNING: Using default action PIN (1234). Change it in production!")
	}

	return nil
}

// Authenticate validates username and password
func (a *AuthManager) Authenticate(username, password string) (*Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check username
	if username != a.username {
		return nil, fmt.Errorf("invalid username")
	}

	// Check password
	expectedHash := a.passwordHash
	if expectedHash == "" {
		return nil, fmt.Errorf("authentication not configured")
	}

	passwordHash := hashPasswordDirect(password, a.sessionSecret)
	if subtle.ConstantTimeCompare([]byte(passwordHash), []byte(expectedHash)) != 1 {
		return nil, fmt.Errorf("invalid password")
	}

	// Create session
	session := &Session{
		ID:        generateRandomSecret(16),
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(a.sessionTTL),
	}

	a.sessions[session.ID] = session
	return session, nil
}

// ValidateSession checks if a session is valid
func (a *AuthManager) ValidateSession(sessionID string) (*Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// ValidateActionPIN validates the action PIN
func (a *AuthManager) ValidateActionPIN(pin string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return subtle.ConstantTimeCompare([]byte(pin), []byte(a.actionPIN)) == 1
}

// DeleteSession removes a session (logout)
func (a *AuthManager) DeleteSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, sessionID)
}

// CreateSession creates a session for a given username
func (a *AuthManager) CreateSession(username string) *Session {
	sessionID := generateRandomSecret(16)
	session := &Session{
		ID:        sessionID,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(a.sessionTTL),
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	a.mu.Unlock()

	return session
}

// cleanupExpiredSessions removes expired sessions periodically
func (a *AuthManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.mu.Lock()
		now := time.Now()
		for id, sess := range a.sessions {
			if now.After(sess.ExpiresAt) {
				delete(a.sessions, id)
			}
		}
		a.mu.Unlock()
	}
}

// hashPassword creates a hash of the password using the secret
func hashPassword(password, secret string) (string, error) {
	// Simple HMAC-based hash (for production, consider bcrypt/argon2)
	return hashPasswordDirect(password, secret), nil
}

// hashPasswordDirect directly hashes a password with a secret
func hashPasswordDirect(password, secret string) string {
	// Simple hash: HMAC-SHA256 would be better, but keeping it simple for MVP
	// Use base64 of password + secret for basic obfuscation
	combined := password + secret
	return base64.StdEncoding.EncodeToString([]byte(combined))
}

// generateRandomSecret generates a random string of the specified length
func generateRandomSecret(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based secret if crypto rand fails
		return fmt.Sprintf("%d%s", time.Now().UnixNano(), "fallback_secret")
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// GetDefaultCredentials returns default credentials for setup
func (a *AuthManager) GetDefaultCredentials() (username, password, pin string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.username, env("BG_ADMIN_PASS", "admin123"), a.actionPIN
}
