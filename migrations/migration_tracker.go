package migrations

import (
	"time"

	"gorm.io/gorm"
)

// schemaMigration tracks which named migrations have been applied.
// The table is bootstrapped automatically by EnsureMigrationsTable.
type schemaMigration struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"not null;uniqueIndex;size:255"`
	AppliedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// EnsureMigrationsTable creates the schema_migrations table if it does not exist.
// Must be called once before any RunMigration calls.
func EnsureMigrationsTable(db *gorm.DB) error {
	return db.AutoMigrate(&schemaMigration{})
}

// RunMigration runs fn only if the migration named name has not been applied yet,
// and reports whether it ran. fn and the tracking record share one transaction,
// so a failed migration is never recorded (on databases with transactional DDL,
// such as Postgres, its schema changes are rolled back too).
func RunMigration(db *gorm.DB, name string, fn func(*gorm.DB) error) (bool, error) {
	var count int64
	if err := db.Model(&schemaMigration{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil // already applied
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Create(&schemaMigration{Name: name, AppliedAt: time.Now().UTC()}).Error
	})
	return err == nil, err
}

// AppliedMigrations returns the names of all recorded migrations, oldest first.
func AppliedMigrations(db *gorm.DB) ([]string, error) {
	var names []string
	err := db.Model(&schemaMigration{}).Order("id").Pluck("name", &names).Error
	return names, err
}
