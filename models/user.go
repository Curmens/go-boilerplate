// models/models.go
package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	BaseModel
	UserID    uint      `gorm:"not null" json:"user_id"`
	Token     string    `gorm:"not null;uniqueIndex;size:255" json:"token" validate:"required,max=255"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	IsRevoked bool      `gorm:"not null;default:false" json:"is_revoked"`

	// Relationships
	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName returns the table name for RefreshToken model
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired checks if the token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(rt.ExpiresAt)
}

// IsValid checks if the token is valid (not expired and not revoked)
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked
}

// AuditLog represents an audit log entry
type AuditLog struct {
	BaseModel
	UserID     uint   `gorm:"not null" json:"user_id"`
	Resource   string `gorm:"not null;size:255" json:"resource" validate:"required,max=255"`
	ResourceID uint   `gorm:"not null" json:"resource_id"`
	Action     string `gorm:"not null;size:255" json:"action" validate:"required,max=255"`
	Details    string `gorm:"size:500" json:"details" validate:"max=500"`

	// Relationships
	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user,omitempty"`
}

// TableName returns the table name for AuditLog model
func (AuditLog) TableName() string {
	return "audit_logs"
}

// User represents a user in the system
type User struct {
	BaseModel
	Email       string     `gorm:"uniqueIndex;not null;size:255" json:"email" validate:"required,email,max=255"`
	Password    string     `gorm:"column:password_hash;not null;size:255" json:"-"`
	FirstName   string     `gorm:"not null;size:100" json:"first_name" validate:"required,min=2,max=100"`
	LastName    string     `gorm:"not null;size:100" json:"last_name" validate:"required,min=2,max=100"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	IsVerified  bool       `gorm:"not null;default:false" json:"is_verified"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	// Foreign Keys
	RoleID uint `gorm:"not null;default:1" json:"role_id"` // Default to basic user role

	// Relationships
	Role          Role           `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"role,omitempty"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	AuditLogs     []AuditLog     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	PhoneNumber   string         `gorm:"size:20" json:"phone_number,omitempty" validate:"omitempty,e164"`
	DateOfBirth   *time.Time     `json:"date_of_birth,omitempty"`
}

// TableName returns the table name for User model
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook executed before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Call base model BeforeCreate
	if err := u.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	// Normalize email
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))

	// Trim and validate names
	u.FirstName = strings.TrimSpace(u.FirstName)
	u.LastName = strings.TrimSpace(u.LastName)

	if u.FirstName == "" || u.LastName == "" {
		return errors.New("first name and last name are required")
	}

	// Hash password if provided
	if u.Password != "" {
		return u.HashPassword()
	}

	return nil
}

// BeforeUpdate hook executed before updating a user
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Call base model BeforeUpdate
	if err := u.BaseModel.BeforeUpdate(tx); err != nil {
		return err
	}

	// Normalize email if changed
	if tx.Statement.Changed("Email") {
		u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	}

	// Trim names if changed
	if tx.Statement.Changed("FirstName") {
		u.FirstName = strings.TrimSpace(u.FirstName)
		if u.FirstName == "" {
			return errors.New("first name cannot be empty")
		}
	}

	if tx.Statement.Changed("LastName") {
		u.LastName = strings.TrimSpace(u.LastName)
		if u.LastName == "" {
			return errors.New("last name cannot be empty")
		}
	}

	// Hash password if changed
	if tx.Statement.Changed("Password") && u.Password != "" {
		return u.HashPassword()
	}

	return nil
}

// HashPassword hashes the user's password using bcrypt
func (u *User) HashPassword() error {
	if u.Password == "" {
		return errors.New("password cannot be empty")
	}

	// Check minimum password length
	if len(u.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	u.Password = string(hashedPassword)
	return nil
}

// SetPassword sets and hashes a new password
func (u *User) SetPassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	u.Password = password
	return u.HashPassword()
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	if password == "" || u.Password == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// ComparePassword is an alias for CheckPassword (for backward compatibility)
func (u *User) ComparePassword(password string) error {
	if !u.CheckPassword(password) {
		return errors.New("invalid password")
	}
	return nil
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// GetDisplayName returns the user's display name (full name or email)
func (u *User) GetDisplayName() string {
	fullName := u.GetFullName()
	if fullName != "" {
		return fullName
	}
	return u.Email
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role.Name == "admin" || u.Role.Name == "super_admin"
}

// HasRole checks if user has a specific role
func (u *User) HasRole(roleName string) bool {
	return strings.EqualFold(u.Role.Name, roleName)
}

// HasPermission checks if user has a specific permission through their role
func (u *User) HasPermission(permissionName string) bool {
	for _, permission := range u.Role.Permissions {
		if strings.EqualFold(permission.Name, permissionName) {
			return true
		}
	}
	return false
}

// CanAccess checks if user can access a resource with a specific action
func (u *User) CanAccess(resource, action string) bool {
	for _, permission := range u.Role.Permissions {
		if strings.EqualFold(permission.Resource, resource) &&
			strings.EqualFold(permission.Action, action) {
			return true
		}
	}
	return false
}

// SetRole sets the user's role by role ID
func (u *User) SetRole(roleID uint) {
	u.RoleID = roleID
}

// SetRoleByName sets the user's role by role name (requires database query)
func (u *User) SetRoleByName(tx *gorm.DB, roleName string) error {
	var role Role
	if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("role '%s' not found: %w", roleName, err)
	}

	u.RoleID = role.ID
	u.Role = role
	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (u *User) UpdateLastLogin(tx *gorm.DB) error {
	now := time.Now().UTC()
	u.LastLoginAt = &now

	return tx.Model(u).Update("last_login_at", now).Error
}

// Deactivate deactivates the user account
func (u *User) Deactivate() {
	u.IsActive = false
}

// Activate activates the user account
func (u *User) Activate() {
	u.IsActive = true
}

// Verify marks the user's email as verified
func (u *User) Verify() {
	u.IsVerified = true
}

// ToPublicJSON returns user data safe for public consumption
func (u *User) ToPublicJSON() map[string]interface{} {
	return map[string]interface{}{
		"id":            u.ID,
		"email":         u.Email,
		"first_name":    u.FirstName,
		"last_name":     u.LastName,
		"full_name":     u.GetFullName(),
		"is_active":     u.IsActive,
		"is_verified":   u.IsVerified,
		"role":          u.Role.Name,
		"last_login_at": u.LastLoginAt,
		"created_at":    u.CreatedAt,
		"updated_at":    u.UpdatedAt,
	}
}
