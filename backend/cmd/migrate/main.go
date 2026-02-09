package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Migration struct {
	Name    string
	UpSQL   string
	DownSQL string
	Version int
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	// Parse command line flags
	action := flag.String("action", "up", "Migration action: up, down, status, create")
	steps := flag.Int("steps", 0, "Number of migrations to run (0 = all)")
	migrationName := flag.String("name", "", "Name for new migration (only for create action)")
	flag.Parse()

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database connection: %v", closeErr)
		}
	}()

	if err := db.Ping(); err != nil {
		_ = db.Close()
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create migrations table if not exists
	if err := createMigrationsTable(db); err != nil {
		_ = db.Close()
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	switch *action {
	case "up":
		if err := runMigrationsUp(db, *steps); err != nil {
			_ = db.Close()
			log.Fatalf("Migration up failed: %v", err)
		}
	case "down":
		if err := runMigrationsDown(db, *steps); err != nil {
			_ = db.Close()
			log.Fatalf("Migration down failed: %v", err)
		}
	case "status":
		if err := showMigrationStatus(db); err != nil {
			_ = db.Close()
			log.Fatalf("Failed to show status: %v", err)
		}
	case "create":
		if *migrationName == "" {
			log.Fatal("Migration name is required for create action. Use -name flag")
		}
		if err := createMigration(*migrationName); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
	default:
		_ = db.Close()
		log.Fatalf("Unknown action: %s. Use: up, down, status, create", *action)
	}
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

func loadMigrations() ([]Migration, error) {
	migrationsDir := "migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %v", err)
	}

	migrationMap := make(map[int]*Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse filename: 000001_init_schema.up.sql or 000001_init_schema.down.sql
		var version int
		var migName, direction string

		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.SplitN(name, "_", 2)
			if len(parts) != 2 {
				continue
			}
			if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
				continue
			}
			migName = strings.TrimSuffix(parts[1], ".up.sql")
			direction = "up"
		} else if strings.HasSuffix(name, ".down.sql") {
			parts := strings.SplitN(name, "_", 2)
			if len(parts) != 2 {
				continue
			}
			if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
				continue
			}
			migName = strings.TrimSuffix(parts[1], ".down.sql")
			direction = "down"
		} else {
			continue
		}

		if migrationMap[version] == nil {
			migrationMap[version] = &Migration{
				Version: version,
				Name:    migName,
			}
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, name)) //nolint:gosec // G304: Migration files are from controlled directory
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", name, err)
		}

		if direction == "up" {
			migrationMap[version].UpSQL = string(content)
		} else {
			migrationMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, mig := range migrationMap {
		migrations = append(migrations, *mig)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func getAppliedMigrations(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func runMigrationsUp(db *sql.DB, steps int) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	count := 0
	for _, mig := range migrations {
		if applied[mig.Version] {
			continue
		}

		if steps > 0 && count >= steps {
			break
		}

		log.Printf("Applying migration %d: %s", mig.Version, mig.Name)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		if _, err := tx.Exec(mig.UpSQL); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Warning: failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to apply migration %d: %v", mig.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", mig.Version); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Warning: failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to record migration %d: %v", mig.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %v", mig.Version, err)
		}

		log.Printf("✓ Migration %d applied successfully", mig.Version)
		count++
	}

	if count == 0 {
		log.Println("No migrations to apply")
	} else {
		log.Printf("Applied %d migration(s)", count)
	}

	return nil
}

func runMigrationsDown(db *sql.DB, steps int) error {
	if steps == 0 {
		steps = 1 // Default to rolling back 1 migration
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Reverse order for rollback
	count := 0
	for i := len(migrations) - 1; i >= 0; i-- {
		mig := migrations[i]

		if !applied[mig.Version] {
			continue
		}

		if count >= steps {
			break
		}

		log.Printf("Rolling back migration %d: %s", mig.Version, mig.Name)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		if _, err := tx.Exec(mig.DownSQL); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Warning: failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to rollback migration %d: %v", mig.Version, err)
		}

		if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", mig.Version); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Warning: failed to rollback transaction: %v", rollbackErr)
			}
			return fmt.Errorf("failed to remove migration record %d: %v", mig.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit rollback %d: %v", mig.Version, err)
		}

		log.Printf("✓ Migration %d rolled back successfully", mig.Version)
		count++
	}

	if count == 0 {
		log.Println("No migrations to rollback")
	} else {
		log.Printf("Rolled back %d migration(s)", count)
	}

	return nil
}

func showMigrationStatus(db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	fmt.Println("\nMigration Status:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-10s %-10s %-30s\n", "Version", "Status", "Name")
	fmt.Println(strings.Repeat("-", 60))

	for _, mig := range migrations {
		status := "pending"
		if applied[mig.Version] {
			status = "applied"
		}
		fmt.Printf("%-10d %-10s %-30s\n", mig.Version, status, mig.Name)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total: %d migrations, %d applied, %d pending\n\n",
		len(migrations), len(applied), len(migrations)-len(applied))

	return nil
}

func createMigration(name string) error {
	migrationsDir := "migrations"

	// Get next version number
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	maxVersion := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		var version int
		if _, err := fmt.Sscanf(file.Name(), "%d", &version); err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	newVersion := maxVersion + 1

	// Create up and down files
	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%06d_%s.up.sql", newVersion, name))
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%06d_%s.down.sql", newVersion, name))

	upTemplate := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n-- Add your SQL here\n", name, "")
	downTemplate := fmt.Sprintf("-- Rollback: %s\n\n-- Add your rollback SQL here\n", name)

	if err := os.WriteFile(upFile, []byte(upTemplate), 0o600); err != nil {
		return fmt.Errorf("failed to create up migration: %v", err)
	}

	if err := os.WriteFile(downFile, []byte(downTemplate), 0o600); err != nil {
		return fmt.Errorf("failed to create down migration: %v", err)
	}

	log.Printf("Created migration files:")
	log.Printf("  - %s", upFile)
	log.Printf("  - %s", downFile)

	return nil
}
