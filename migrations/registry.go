package migrations

import "gorm.io/gorm"

// Migration is a named, run-once schema change.
type Migration struct {
	Name string
	Up   func(*gorm.DB) error
}

// All returns every migration in the order it must be applied. Names are
// recorded in schema_migrations, so never rename or reorder an applied one —
// append new migrations to the end instead.
//
// Example:
//
//	{Name: "create_users_table", Up: func(db *gorm.DB) error {
//		return db.AutoMigrate(&models.User{})
//	}},
func All() []Migration {
	return []Migration{}
}

// Apply bootstraps the tracking table and runs every pending migration in order.
func Apply(db *gorm.DB, onApplied func(name string)) error {
	if err := EnsureMigrationsTable(db); err != nil {
		return err
	}

	for _, m := range All() {
		applied, err := RunMigration(db, m.Name, m.Up)
		if err != nil {
			return &Error{Name: m.Name, Err: err}
		}
		if applied && onApplied != nil {
			onApplied(m.Name)
		}
	}
	return nil
}

// Error identifies which migration failed.
type Error struct {
	Name string
	Err  error
}

func (e *Error) Error() string { return "migration " + e.Name + " failed: " + e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }
