package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"example.com/config"
	"example.com/database"
	"example.com/migrations"
	"example.com/seeder"
	"example.com/utils"
)

func main() {
	var (
		up        = flag.Bool("up", false, "Apply pending migrations")
		status    = flag.Bool("status", false, "List applied migrations")
		seed      = flag.Bool("seed", false, "Run all seeders")
		seedOnly  = flag.String("seed-only", "", "Run specific seeders (comma-separated)")
		listSeeds = flag.Bool("list-seeders", false, "List available seeders")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := utils.NewLogger(cfg.Logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	seederInstance := seeder.New(db.DB, logger)
	ctx := context.Background()

	switch {
	case *up:
		if err := db.Migrate(); err != nil {
			exit(err)
		}

	case *status:
		if err := migrations.EnsureMigrationsTable(db.DB); err != nil {
			exit(err)
		}
		applied, err := migrations.AppliedMigrations(db.DB)
		if err != nil {
			exit(err)
		}
		done := make(map[string]bool, len(applied))
		for _, name := range applied {
			done[name] = true
		}
		for _, m := range migrations.All() {
			state := "pending"
			if done[m.Name] {
				state = "applied"
			}
			fmt.Printf("  [%s] %s\n", state, m.Name)
		}

	case *listSeeds:
		fmt.Println("Available seeders:")
		for _, name := range seederInstance.ListAvailableSeeders() {
			fmt.Printf("  - %s\n", name)
		}

	case *seedOnly != "":
		names := strings.Split(*seedOnly, ",")
		for i, name := range names {
			names[i] = strings.TrimSpace(name)
		}
		if err := seederInstance.SeedSpecific(ctx, names); err != nil {
			exit(err)
		}

	case *seed:
		if err := seederInstance.SeedAll(ctx); err != nil {
			exit(err)
		}

	default:
		flag.Usage()
		fmt.Println("\nExamples:")
		fmt.Println("  go run ./cmd/migrate -up                     # Apply pending migrations")
		fmt.Println("  go run ./cmd/migrate -status                 # Show migration status")
		fmt.Println("  go run ./cmd/migrate -seed                   # Run all seeders")
		fmt.Println("  go run ./cmd/migrate -seed-only=roles,users  # Run specific seeders")
	}
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
