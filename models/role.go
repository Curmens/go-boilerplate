package models

import (
	"gorm.io/gorm"
	"strings"
)

// Role represents a role in the system
type Role struct {
	BaseModel
	Name        string `gorm:"uniqueIndex;not null;size:100" json:"name" validate:"required,min=2,max=100"`
	Description string `gorm:"size:500" json:"description" validate:"max=500"`
	IsActive    bool   `gorm:"not null;default:true" json:"is_active"`

	// Relationships
	Users       []User       `gorm:"foreignKey:RoleID" json:"users,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"permissions,omitempty"`
}

// TableName returns the table name for Role model
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate hook executed before creating a role
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if err := r.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	r.Description = strings.TrimSpace(r.Description)

	return nil
}

// BeforeUpdate hook executed before updating a role
func (r *Role) BeforeUpdate(tx *gorm.DB) error {
	if err := r.BaseModel.BeforeUpdate(tx); err != nil {
		return err
	}

	if tx.Statement.Changed("Name") {
		r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	}

	if tx.Statement.Changed("Description") {
		r.Description = strings.TrimSpace(r.Description)
	}

	return nil
}

// HasPermission checks if role has a specific permission
func (r *Role) HasPermission(permissionName string) bool {
	for _, permission := range r.Permissions {
		if strings.EqualFold(permission.Name, permissionName) {
			return true
		}
	}
	return false
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(tx *gorm.DB, permission *Permission) error {
	return tx.Model(r).Association("Permissions").Append(permission)
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(tx *gorm.DB, permission *Permission) error {
	return tx.Model(r).Association("Permissions").Delete(permission)
}
