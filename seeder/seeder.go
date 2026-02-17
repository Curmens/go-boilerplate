package seeder

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/models"
	"example.com/utils"
	"gorm.io/gorm"
)

// Seeder handles database seeding operations
type Seeder struct {
	db      *gorm.DB
	logger  *utils.Logger
	seeders map[string]SeederFunc
	order   []string
}

// SeederFunc is a function that seeds data
type SeederFunc func(ctx context.Context, db *gorm.DB) error

// New creates a new Seeder instance
func New(db *gorm.DB, logger *utils.Logger) *Seeder {
	s := &Seeder{
		db:      db,
		logger:  logger,
		seeders: make(map[string]SeederFunc),
		order:   []string{},
	}

	// Register seeders in order of dependency
	s.Register("permissions", seedPermissions)
	s.Register("roles", seedRoles)
	s.Register("users", seedUsers)

	return s
}

// Register adds a seeder function
func (s *Seeder) Register(name string, fn SeederFunc) {
	s.seeders[name] = fn
	s.order = append(s.order, name)
}

// ListAvailableSeeders returns a list of available seeder names
func (s *Seeder) ListAvailableSeeders() []string {
	return s.order
}

// SeedAll runs all seeders in order
func (s *Seeder) SeedAll(ctx context.Context) error {
	s.logger.Info("Starting all seeders", map[string]interface{}{
		"count": len(s.order),
	})

	for _, name := range s.order {
		if err := s.runSeeder(ctx, name); err != nil {
			return fmt.Errorf("seeder '%s' failed: %w", name, err)
		}
	}

	s.logger.Info("All seeders completed successfully", nil)
	return nil
}

// SeedSpecific runs specific seeders by name
func (s *Seeder) SeedSpecific(ctx context.Context, names []string) error {
	for _, name := range names {
		if _, exists := s.seeders[name]; !exists {
			return fmt.Errorf("seeder '%s' not found", name)
		}
	}

	for _, name := range names {
		if err := s.runSeeder(ctx, name); err != nil {
			return fmt.Errorf("seeder '%s' failed: %w", name, err)
		}
	}

	return nil
}

// runSeeder executes a single seeder
func (s *Seeder) runSeeder(ctx context.Context, name string) error {
	fn, exists := s.seeders[name]
	if !exists {
		return fmt.Errorf("seeder '%s' not found", name)
	}

	s.logger.Info("Running seeder", map[string]interface{}{
		"name": name,
	})

	if err := fn(ctx, s.db); err != nil {
		s.logger.Error("Seeder failed", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
		return err
	}

	s.logger.Info("Seeder completed", map[string]interface{}{
		"name": name,
	})

	return nil
}

// ResetDatabase drops all tables and re-runs migrations and seeders
func (s *Seeder) ResetDatabase(ctx context.Context) error {
	s.logger.Warn("Resetting database - all data will be lost", nil)

	// Get all table names
	var tables []string
	if err := s.db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		// Try MySQL syntax
		if err := s.db.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
			return fmt.Errorf("failed to get table list: %w", err)
		}
	}

	// Drop tables in reverse order to handle foreign keys
	sort.Sort(sort.Reverse(sort.StringSlice(tables)))
	for _, table := range tables {
		if table == "schema_migrations" || table == "gorm_migrations" {
			continue
		}
		s.logger.Info("Dropping table", map[string]interface{}{"table": table})
		if err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			s.logger.Warn("Failed to drop table", map[string]interface{}{
				"table": table,
				"error": err.Error(),
			})
		}
	}

	s.logger.Info("Database reset completed", nil)
	return nil
}

// CleanupTestData removes test data (users with email containing 'test')
func (s *Seeder) CleanupTestData(ctx context.Context) error {
	s.logger.Info("Cleaning up test data", nil)

	// Delete test users
	result := s.db.Where("email LIKE ?", "%test%").Delete(&models.User{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete test users: %w", result.Error)
	}

	s.logger.Info("Test data cleanup completed", map[string]interface{}{
		"deleted_users": result.RowsAffected,
	})

	return nil
}

// seedPermissions creates default permissions
func seedPermissions(ctx context.Context, db *gorm.DB) error {
	permissions := []models.Permission{
		// User permissions
		{Name: "users:read", Description: "Read user data", Resource: "users", Action: "read"},
		{Name: "users:create", Description: "Create users", Resource: "users", Action: "create"},
		{Name: "users:update", Description: "Update users", Resource: "users", Action: "update"},
		{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete"},

		// Role permissions
		{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read"},
		{Name: "roles:create", Description: "Create roles", Resource: "roles", Action: "create"},
		{Name: "roles:update", Description: "Update roles", Resource: "roles", Action: "update"},
		{Name: "roles:delete", Description: "Delete roles", Resource: "roles", Action: "delete"},

		// Permission management
		{Name: "permissions:read", Description: "Read permissions", Resource: "permissions", Action: "read"},
		{Name: "permissions:manage", Description: "Manage permissions", Resource: "permissions", Action: "manage"},

		// System permissions
		{Name: "system:admin", Description: "Full system access", Resource: "system", Action: "admin"},
		{Name: "system:settings", Description: "Manage system settings", Resource: "system", Action: "settings"},
	}

	for _, perm := range permissions {
		existing := models.Permission{}
		result := db.Where("name = ?", perm.Name).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&perm).Error; err != nil {
				return fmt.Errorf("failed to create permission '%s': %w", perm.Name, err)
			}
		}
	}

	return nil
}

// seedRoles creates default roles with permissions
func seedRoles(ctx context.Context, db *gorm.DB) error {
	// Define roles with their permissions
	rolesWithPermissions := map[string]struct {
		Description string
		Permissions []string
	}{
		"user": {
			Description: "Standard user role",
			Permissions: []string{"users:read"},
		},
		"moderator": {
			Description: "Moderator role with extended permissions",
			Permissions: []string{"users:read", "users:update"},
		},
		"admin": {
			Description: "Administrator role",
			Permissions: []string{
				"users:read", "users:create", "users:update", "users:delete",
				"roles:read", "permissions:read",
			},
		},
		"super_admin": {
			Description: "Super administrator with full access",
			Permissions: []string{
				"users:read", "users:create", "users:update", "users:delete",
				"roles:read", "roles:create", "roles:update", "roles:delete",
				"permissions:read", "permissions:manage",
				"system:admin", "system:settings",
			},
		},
	}

	for roleName, config := range rolesWithPermissions {
		// Create or find role
		role := models.Role{}
		result := db.Where("name = ?", roleName).First(&role)
		if result.Error == gorm.ErrRecordNotFound {
			role = models.Role{
				Name:        roleName,
				Description: config.Description,
				IsActive:    true,
			}
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("failed to create role '%s': %w", roleName, err)
			}
		}

		// Assign permissions
		var permissions []models.Permission
		if err := db.Where("name IN ?", config.Permissions).Find(&permissions).Error; err != nil {
			return fmt.Errorf("failed to find permissions for role '%s': %w", roleName, err)
		}

		if err := db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			return fmt.Errorf("failed to assign permissions to role '%s': %w", roleName, err)
		}
	}

	return nil
}

// seedUsers creates default users
func seedUsers(ctx context.Context, db *gorm.DB) error {
	// Find roles
	var adminRole, userRole models.Role
	if err := db.Where("name = ?", "super_admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("super_admin role not found: %w", err)
	}
	if err := db.Where("name = ?", "user").First(&userRole).Error; err != nil {
		return fmt.Errorf("user role not found: %w", err)
	}

	// Default users
	users := []struct {
		Email     string
		Password  string
		FirstName string
		LastName  string
		RoleID    uint
	}{
		{
			Email:     "admin@example.com",
			Password:  "Admin123!", // Should be changed in production
			FirstName: "System",
			LastName:  "Admin",
			RoleID:    adminRole.ID,
		},
		{
			Email:     "user@example.com",
			Password:  "User123!", // Should be changed in production
			FirstName: "Default",
			LastName:  "User",
			RoleID:    userRole.ID,
		},
	}

	for _, u := range users {
		existing := models.User{}
		result := db.Where("email = ?", strings.ToLower(u.Email)).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			user := models.User{
				Email:      u.Email,
				Password:   u.Password,
				FirstName:  u.FirstName,
				LastName:   u.LastName,
				RoleID:     u.RoleID,
				IsActive:   true,
				IsVerified: true,
			}
			if err := db.Create(&user).Error; err != nil {
				return fmt.Errorf("failed to create user '%s': %w", u.Email, err)
			}
		}
	}

	return nil
}
