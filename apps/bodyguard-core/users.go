package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// UserRole defines the role of a user
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleGuest UserRole = "guest"
)

// User represents a system user
type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	PasswordHash   string    `json:"-"` // Never expose in JSON
	Role           UserRole  `json:"role"`
	ProfilePicture string    `json:"profile_picture"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"`
	LastLogin      time.Time `json:"last_login"`
	Active         bool      `json:"active"`
}

// UserManager manages multiple users
type UserManager struct {
	mu          sync.RWMutex
	users       map[string]*User
	sessionSecret string
}

// NewUserManager creates a new user manager
func NewUserManager(sessionSecret string) *UserManager {
	return &UserManager{
		users:         make(map[string]*User),
		sessionSecret: sessionSecret,
	}
}

// Initialize creates default admin user if no users exist
func (um *UserManager) Initialize(defaultUsername, defaultPassword string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if len(um.users) == 0 {
		// Hash the password
		passwordHash := hashPasswordDirect(defaultPassword, um.sessionSecret)

		// Create default admin user
		adminUser := &User{
			ID:           generateUserID(),
			Username:     defaultUsername,
			PasswordHash: passwordHash,
			Role:         RoleAdmin,
			CreatedAt:    time.Now(),
			Active:       true,
		}
		um.users[adminUser.ID] = adminUser
		fmt.Printf("✅ Created default admin user: %s (ID: %s)\n", defaultUsername, adminUser.ID)
	}

	return nil
}

// ListUsers returns all users (without password hashes)
func (um *UserManager) ListUsers() []User {
	um.mu.RLock()
	defer um.mu.RUnlock()

	users := make([]User, 0, len(um.users))
	for _, user := range um.users {
		users = append(users, User{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			CreatedBy: user.CreatedBy,
			LastLogin: user.LastLogin,
			Active:    user.Active,
		})
	}
	return users
}

// GetUserByUsername finds a user by username
func (um *UserManager) GetUserByUsername(username string) (*User, bool) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	for _, user := range um.users {
		if user.Username == username {
			return user, true
		}
	}
	return nil, false
}

// GetUserByID finds a user by ID
func (um *UserManager) GetUserByID(id string) (*User, bool) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.users[id]
	if !exists {
		return nil, false
	}
	return user, true
}

// AddUser adds a new user
func (um *UserManager) AddUser(username, password string, role UserRole, createdBy string) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	// Check if username already exists
	for _, user := range um.users {
		if user.Username == username {
			return nil, fmt.Errorf("username already exists")
		}
	}

	// Hash password
	passwordHash := hashPasswordDirect(password, um.sessionSecret)

	// Create new user
	user := &User{
		ID:           generateUserID(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
		Active:       true,
	}

	um.users[user.ID] = user
	return user, nil
}

// DeleteUser removes a user
func (um *UserManager) DeleteUser(id, requestingUserID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Prevent self-deletion
	if id == requestingUserID {
		return fmt.Errorf("cannot delete your own account")
	}

	// Prevent deleting the last admin
	if user.Role == RoleAdmin {
		adminCount := 0
		for _, u := range um.users {
			if u.Role == RoleAdmin && u.Active {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot delete the last admin user")
		}
	}

	delete(um.users, id)
	return nil
}

// UpdateUserPassword changes a user's password
func (um *UserManager) UpdateUserPassword(id, newPassword string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	user.PasswordHash = hashPasswordDirect(newPassword, um.sessionSecret)
	return nil
}

// UpdateUserRole changes a user's role
func (um *UserManager) UpdateUserRole(id, requestingUserID string, newRole UserRole) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Prevent self-role-change
	if id == requestingUserID {
		return fmt.Errorf("cannot change your own role")
	}

	// Prevent removing admin role from the last admin
	if user.Role == RoleAdmin && newRole != RoleAdmin {
		adminCount := 0
		for _, u := range um.users {
			if u.Role == RoleAdmin && u.Active {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot change role of the last admin user")
		}
	}

	user.Role = newRole
	return nil
}

// SetUserActive sets a user's active status
func (um *UserManager) SetUserActive(id, requestingUserID string, active bool) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Prevent self-deactivation
	if id == requestingUserID && !active {
		return fmt.Errorf("cannot deactivate your own account")
	}

	// Prevent deactivating the last admin
	if user.Role == RoleAdmin && !active {
		adminCount := 0
		for _, u := range um.users {
			if u.Role == RoleAdmin && u.Active {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot deactivate the last admin user")
		}
	}

	user.Active = active
	return nil
}

// Authenticate validates credentials and returns the user
func (um *UserManager) Authenticate(username, password string) (*User, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.GetUserByUsername(username)
	if !exists {
		return nil, fmt.Errorf("invalid username or password")
	}

	if !user.Active {
		return nil, fmt.Errorf("user account is disabled")
	}

	passwordHash := hashPasswordDirect(password, um.sessionSecret)
	if subtle.ConstantTimeCompare([]byte(passwordHash), []byte(user.PasswordHash)) != 1 {
		return nil, fmt.Errorf("invalid username or password")
	}

	return user, nil
}

// UpdateLastLogin updates the last login time for a user
func (um *UserManager) UpdateLastLogin(userID string) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.users[userID]; exists {
		user.LastLogin = time.Now()
	}
}

// UpdateProfile updates user's profile information
func (um *UserManager) UpdateProfile(userID, fullName, email string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	user.FullName = fullName
	user.Email = email
	return nil
}

// UpdateProfilePicture updates user's profile picture (base64 or URL)
func (um *UserManager) UpdateProfilePicture(userID, profilePicture string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	user.ProfilePicture = profilePicture
	return nil
}

// GetProfilePicture returns user's profile picture URL
func (um *UserManager) GetProfilePicture(userID string) string {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.users[userID]
	if !exists {
		return ""
	}

	if user.ProfilePicture != "" {
		return user.ProfilePicture
	}

	// Return default avatar based on username
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=3b82f6&color=fff", user.Username)
}

// generateUserID generates a unique user ID
func generateUserID() string {
	return fmt.Sprintf("usr-%d", time.Now().UnixNano())
}

// HTTP Handlers for user management

// handleListUsers returns all users
func (app *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Require auth
	session := app.requireAuth(w, r)
	if session == nil {
		return
	}

	users := app.userManager.ListUsers()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// handleAddUser creates a new user
func (app *App) handleAddUser(w http.ResponseWriter, r *http.Request) {
	// Require admin
	session := app.requireAdmin(w, r)
	if session == nil {
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "Username is required")
		return
	}
	if len(req.Password) < 6 {
		respondError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	role := RoleUser
	if req.Role == string(RoleAdmin) || req.Role == string(RoleGuest) {
		role = UserRole(req.Role)
	}

	user, err := app.userManager.AddUser(req.Username, req.Password, role, session.Username)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"user": User{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			CreatedBy: user.CreatedBy,
			Active:    user.Active,
		},
	})
}

// handleDeleteUser removes a user
func (app *App) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Require admin
	session := app.requireAdmin(w, r)
	if session == nil {
		return
	}

	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	// Get requesting user ID from session
	requestingUserID := ""
	for _, u := range app.userManager.ListUsers() {
		if u.Username == session.Username {
			requestingUserID = u.ID
			break
		}
	}

	if err := app.userManager.DeleteUser(userID, requestingUserID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
	})
}

// handleUpdateUser updates a user
func (app *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Require admin
	session := app.requireAdmin(w, r)
	if session == nil {
		return
	}

	userID := chi.URLParam(r, "id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var req struct {
		Role   *string `json:"role"`
		Active *bool   `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get requesting user ID
	requestingUserID := ""
	for _, u := range app.userManager.ListUsers() {
		if u.Username == session.Username {
			requestingUserID = u.ID
			break
		}
	}

	// Update role if provided
	if req.Role != nil {
		role := UserRole(*req.Role)
		if err := app.userManager.UpdateUserRole(userID, requestingUserID, role); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Update active status if provided
	if req.Active != nil {
		if err := app.userManager.SetUserActive(userID, requestingUserID, *req.Active); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
	})
}

// handleChangePassword changes a user's password
func (app *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	// Require auth
	session := app.requireAuth(w, r)
	if session == nil {
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.NewPassword) < 6 {
		respondError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Verify current password
	user, err := app.userManager.Authenticate(session.Username, req.CurrentPassword)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Update password
	if err := app.userManager.UpdateUserPassword(user.ID, req.NewPassword); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password changed successfully",
	})
}

// handleGetCurrentUser returns the current user's profile
func (app *App) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Require auth
	session := app.requireAuth(w, r)
	if session == nil {
		return
	}

	user, exists := app.userManager.GetUserByUsername(session.Username)
	if !exists {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Add profile picture URL
	profilePicture := app.userManager.GetProfilePicture(user.ID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":              user.ID,
			"username":        user.Username,
			"role":            user.Role,
			"full_name":       user.FullName,
			"email":           user.Email,
			"profile_picture": profilePicture,
			"created_at":      user.CreatedAt,
			"last_login":      user.LastLogin,
			"active":          user.Active,
		},
	})
}

// handleUpdateProfile updates current user's profile
func (app *App) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Require auth
	session := app.requireAuth(w, r)
	if session == nil {
		return
	}

	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user ID
	user, exists := app.userManager.GetUserByUsername(session.Username)
	if !exists {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Update profile
	if err := app.userManager.UpdateProfile(user.ID, req.FullName, req.Email); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Profile updated successfully",
	})
}

// handleUpdateProfilePicture updates current user's profile picture
func (app *App) handleUpdateProfilePicture(w http.ResponseWriter, r *http.Request) {
	// Require auth
	session := app.requireAuth(w, r)
	if session == nil {
		return
	}

	var req struct {
		ProfilePicture string `json:"profile_picture"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate profile picture (basic validation for URL or base64)
	if req.ProfilePicture == "" {
		respondError(w, http.StatusBadRequest, "Profile picture is required")
		return
	}

	// Check if it's a valid URL or base64
	isValid := false
	if len(req.ProfilePicture) > 1000 {
		// Assume base64 if very long
		isValid = true
	} else if strings.HasPrefix(req.ProfilePicture, "http://") || strings.HasPrefix(req.ProfilePicture, "https://") {
		isValid = true
	}

	if !isValid {
		respondError(w, http.StatusBadRequest, "Profile picture must be a valid URL or base64 encoded image")
		return
	}

	// Get user ID
	user, exists := app.userManager.GetUserByUsername(session.Username)
	if !exists {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Update profile picture
	if err := app.userManager.UpdateProfilePicture(user.ID, req.ProfilePicture); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Profile picture updated successfully",
		"profile_picture": app.userManager.GetProfilePicture(user.ID),
	})
}

// handleLogin authenticates user and creates session
func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Authenticate user using UserManager
	user, err := app.userManager.Authenticate(req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Create session using AuthManager
	session := app.auth.CreateSession(user.Username)

	// Update last login
	app.userManager.UpdateLastLogin(user.ID)

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"session_id": session.ID,
		"user": map[string]interface{}{
			"username": session.Username,
		},
	})
}

// handleLogout clears the session
func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get current session
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		// Invalidate session
		app.auth.DeleteSession(sessionCookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

// requireAuth checks for valid session and returns it
func (app *App) requireAuth(w http.ResponseWriter, r *http.Request) *Session {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return nil
	}

	session, err := app.auth.ValidateSession(sessionCookie.Value)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid session")
		return nil
	}

	return session
}

// requireAdmin checks for valid admin session
func (app *App) requireAdmin(w http.ResponseWriter, r *http.Request) *Session {
	session := app.requireAuth(w, r)
	if session == nil {
		return nil
	}

	// Check if user is admin
	user, exists := app.userManager.GetUserByUsername(session.Username)
	if !exists || user.Role != RoleAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return nil
	}

	return session
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}
