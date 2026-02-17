package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"example.com/config"
	"example.com/database"
	"example.com/seeder"
	"example.com/utils"
)

func main() {
	var (
		up        = flag.Bool("up", false, "Run migrations up")
		down      = flag.Bool("down", false, "Run migrations down")
		seed      = flag.Bool("seed", false, "Run all seeders")
		seedOnly  = flag.String("seed-only", "", "Run specific seeders (comma-separated): permissions,roles,users")
		listSeeds = flag.Bool("list-seeders", false, "List all available seeders")
		resetDB   = flag.Bool("reset", false, "Reset database (DANGEROUS - drops all data)")
		cleanupDB = flag.Bool("cleanup", false, "Cleanup test data")
	)
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := utils.NewLogger(cfg.Logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Connect to database
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Errorf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create seeder instance
	seederInstance := seeder.New(db.DB, logger)

	switch {
	case *up:
		if err := db.Migrate(); err != nil {
			logger.Errorf("Migration failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Migrations completed successfully", nil)

	case *down:
		if err := db.Rollback(); err != nil {
			logger.Errorf("Migration rollback failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Migrations rolled back successfully", nil)

	case *listSeeds:
		availableSeeders := seederInstance.ListAvailableSeeders()
		logger.Info("Available seeders:", map[string]interface{}{
			"seeders": strings.Join(availableSeeders, ", "),
			"count":   len(availableSeeders),
		})
		fmt.Println("\nAvailable seeders:")
		for _, name := range availableSeeders {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println("\nUsage examples:")
		fmt.Println("  go run cmd/migrate/main.go -seed                    # Run all seeders")
		fmt.Println("  go run cmd/migrate/main.go -seed-only=permissions   # Run single seeder")
		fmt.Println("  go run cmd/migrate/main.go -seed-only=roles,users   # Run multiple seeders")

	case *seedOnly != "":
		// Parse comma-separated seeder names
		seederNames := strings.Split(*seedOnly, ",")
		for i, name := range seederNames {
			seederNames[i] = strings.TrimSpace(name)
		}

		logger.Info("Running specific seeders", map[string]interface{}{
			"seeders": seederNames,
		})

		if err := seederInstance.SeedSpecific(context.Background(), seederNames); err != nil {
			logger.Errorf("Seeding failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Specific seeding completed successfully", nil)

	case *seed:
		if err := seederInstance.SeedAll(context.Background()); err != nil {
			logger.Errorf("Seeding failed: %v", err)
			os.Exit(1)
		}
		logger.Info("All seeding completed successfully", nil)

	case *resetDB:
		fmt.Println("\n⚠️  WARNING: This will delete ALL data in the database!")
		fmt.Print("Press Enter to continue or Ctrl+C to cancel...")

		var input string
		fmt.Scanln(&input)

		if err := seederInstance.ResetDatabase(context.Background()); err != nil {
			logger.Errorf("Database reset failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Database reset completed successfully", nil)

	case *cleanupDB:
		if err := seederInstance.CleanupTestData(context.Background()); err != nil {
			logger.Errorf("Cleanup failed: %v", err)
			os.Exit(1)
		}
		logger.Info("Cleanup completed successfully", nil)

	default:
		flag.Usage()
		fmt.Println("\nExamples:")
		fmt.Println("  go run cmd/migrate/main.go -up                         # Run migrations")
		fmt.Println("  go run cmd/migrate/main.go -seed                       # Run all seeders")
		fmt.Println("  go run cmd/migrate/main.go -seed-only=roles            # Run only roles seeder")
		fmt.Println("  go run cmd/migrate/main.go -seed-only=roles,users      # Run multiple seeders")
		fmt.Println("  go run cmd/migrate/main.go -list-seeders               # List available seeders")
		fmt.Println("  go run cmd/migrate/main.go -cleanup                    # Cleanup test data")
		fmt.Println("  go run cmd/migrate/main.go -reset                      # Reset database (DANGEROUS)")
	}
}
