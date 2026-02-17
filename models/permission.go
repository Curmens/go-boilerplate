package models

import (
	"gorm.io/gorm"
	"strings"
)

// Permission represents a permission in the system
type Permission struct {
	BaseModel
	Name        string `gorm:"uniqueIndex;not null;size:100" json:"name" validate:"required,min=2,max=100"`
	Description string `gorm:"size:500" json:"description" validate:"max=500"`
	Resource    string `gorm:"not null;size:100" json:"resource" validate:"required,max=100"`
	Action      string `gorm:"not null;size:100" json:"action" validate:"required,max=100"`

	// Relationships
	Roles []Role `gorm:"many2many:role_permissions" json:"roles,omitempty"`
}

// TableName returns the table name for Permission model
func (Permission) TableName() string {
	return "permissions"
}

// BeforeCreate hook executed before creating a permission
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if err := p.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	p.Name = strings.ToLower(strings.TrimSpace(p.Name))
	p.Resource = strings.ToLower(strings.TrimSpace(p.Resource))
	p.Action = strings.ToLower(strings.TrimSpace(p.Action))
	p.Description = strings.TrimSpace(p.Description)

	return nil
}

// BeforeUpdate hook executed before updating a permission
func (p *Permission) BeforeUpdate(tx *gorm.DB) error {
	if err := p.BaseModel.BeforeUpdate(tx); err != nil {
		return err
	}

	if tx.Statement.Changed("Name") {
		p.Name = strings.ToLower(strings.TrimSpace(p.Name))
	}

	if tx.Statement.Changed("Resource") {
		p.Resource = strings.ToLower(strings.TrimSpace(p.Resource))
	}

	if tx.Statement.Changed("Action") {
		p.Action = strings.ToLower(strings.TrimSpace(p.Action))
	}

	if tx.Statement.Changed("Description") {
		p.Description = strings.TrimSpace(p.Description)
	}

	return nil
}

// GetFullName returns the permission's full name (resource:action)
func (p *Permission) GetFullName() string {
	return p.Resource + ":" + p.Action
}
