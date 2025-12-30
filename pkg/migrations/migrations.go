package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"koteyye_music_be/pkg/database"

	"github.com/jackc/pgx/v5"
)

type Migrator struct {
	db            *database.DB
	logger        *slog.Logger
	migrationsDir string
}

type Migration struct {
	Name      string
	AppliedAt *time.Time
}

func NewMigrator(db *database.DB, logger *slog.Logger) *Migrator {
	// Get the project root directory by going up from this file's location
	// Current file: pkg/migrations/migrations.go
	// Need to go up 3 levels: migrations.go -> migrations -> pkg -> project root
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	migrationsDir := filepath.Join(projectRoot, "migrations")

	return &Migrator{
		db:            db,
		logger:        logger,
		migrationsDir: migrationsDir,
	}
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	files, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	if len(files) == 0 {
		m.logger.Info("No migrations found", "dir", m.migrationsDir)
		return nil
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Apply pending migrations
	appliedCount := 0
	for _, file := range files {
		if _, exists := applied[file]; exists {
			m.logger.Debug("Migration already applied", "file", file)
			continue
		}

		m.logger.Info("Applying migration", "file", file)
		if err := m.applyMigration(ctx, file); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}

		m.logger.Info("Migration applied successfully", "file", file)
		appliedCount++
	}

	if appliedCount > 0 {
		m.logger.Info("Migrations completed", "applied", appliedCount, "total", len(files))
	} else {
		m.logger.Info("All migrations are up to date", "total", len(files))
	}

	return nil
}

// createMigrationsTable creates schema_migrations table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := m.db.Pool.Exec(ctx, query)
	return err
}

// getMigrationFiles returns a sorted list of migration file names
func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Warn("Migrations directory not found", "dir", m.migrationsDir)
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only process .sql files
		if strings.HasSuffix(name, ".sql") {
			files = append(files, name)
		}
	}

	// Sort files alphabetically to ensure correct order
	sort.Strings(files)

	return files, nil
}

// getAppliedMigrations returns a map of applied migration names
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	query := `SELECT name FROM schema_migrations`

	rows, err := m.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, nil
}

// applyMigration applies a single migration file
func (m *Migrator) applyMigration(ctx context.Context, filename string) error {
	// Read migration file content
	content, err := os.ReadFile(filepath.Join(m.migrationsDir, filename))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Begin transaction
	tx, err := m.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	_, err = tx.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	if err := m.recordMigration(ctx, tx, filename); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// recordMigration records a migration as applied
func (m *Migrator) recordMigration(ctx context.Context, tx pgx.Tx, filename string) error {
	query := `
		INSERT INTO schema_migrations (name)
		VALUES ($1)
		ON CONFLICT (name) DO NOTHING
	`

	_, err := tx.Exec(ctx, query, filename)
	return err
}

// GetMigrationStatus returns status of all migrations
func (m *Migrator) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
	files, err := m.getMigrationFiles()
	if err != nil {
		return nil, err
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var status []Migration
	for _, file := range files {
		migration := Migration{
			Name: file,
		}

		// Get applied time if migration was applied
		if _, exists := applied[file]; exists {
			query := `SELECT applied_at FROM schema_migrations WHERE name = $1`
			var appliedAt time.Time
			if err := m.db.Pool.QueryRow(ctx, query, file).Scan(&appliedAt); err == nil {
				migration.AppliedAt = &appliedAt
			}
		}

		status = append(status, migration)
	}

	return status, nil
}
