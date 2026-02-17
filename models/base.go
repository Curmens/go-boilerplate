// models/base.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate sets CreatedAt and UpdatedAt timestamps
func (bm *BaseModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	bm.CreatedAt = now
	bm.UpdatedAt = now
	return nil
}

// BeforeUpdate sets UpdatedAt timestamp
func (bm *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	bm.UpdatedAt = time.Now().UTC()
	return nil
}
