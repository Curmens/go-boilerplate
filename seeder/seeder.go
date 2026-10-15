package seeder

import (
	"context"
	"fmt"

	"example.com/utils"

	"gorm.io/gorm"
)

// SeederFunc is a function that seeds data. Seeders should be idempotent
// (e.g. use FirstOrCreate) so they are safe to re-run.
type SeederFunc func(ctx context.Context, db *gorm.DB) error

// Seeder handles database seeding operations
type Seeder struct {
	db      *gorm.DB
	logger  *utils.Logger
	seeders map[string]SeederFunc
	order   []string
}

// New creates a Seeder with all seeders registered
func New(db *gorm.DB, logger *utils.Logger) *Seeder {
	s := &Seeder{
		db:      db,
		logger:  logger,
		seeders: make(map[string]SeederFunc),
	}

	// Register seeders in dependency order, e.g.:
	//   s.Register("roles", seedRoles)
	//   s.Register("users", seedUsers)

	return s
}

// Register adds a seeder function
func (s *Seeder) Register(name string, fn SeederFunc) {
	if _, exists := s.seeders[name]; !exists {
		s.order = append(s.order, name)
	}
	s.seeders[name] = fn
}

// ListAvailableSeeders returns seeder names in registration order
func (s *Seeder) ListAvailableSeeders() []string {
	return s.order
}

// SeedAll runs all seeders in order
func (s *Seeder) SeedAll(ctx context.Context) error {
	return s.SeedSpecific(ctx, s.order)
}

// SeedSpecific runs the named seeders in the order given
func (s *Seeder) SeedSpecific(ctx context.Context, names []string) error {
	for _, name := range names {
		if _, exists := s.seeders[name]; !exists {
			return fmt.Errorf("unknown seeder %q (available: %v)", name, s.order)
		}
	}

	for _, name := range names {
		s.logger.Info("Running seeder", map[string]interface{}{"seeder": name})
		if err := s.seeders[name](ctx, s.db.WithContext(ctx)); err != nil {
			return fmt.Errorf("seeder %q failed: %w", name, err)
		}
	}
	return nil
}
