package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func RunInitMigration(db *sql.DB, migrationPath string) error {
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	statements := strings.Split(string(sqlBytes), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			if isIgnorableMigrationError(err) {
				continue
			}
			return fmt.Errorf("run migration statement failed: %w", err)
		}
	}
	return nil
}

func isIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	lowerErr := strings.ToLower(err.Error())
	// MySQL compatibility: older versions don't support IF NOT EXISTS on ADD COLUMN.
	// We run init migration on each startup, so duplicated schema objects should be skipped.
	return strings.Contains(lowerErr, "duplicate column name") ||
		strings.Contains(lowerErr, "duplicate key name")
}
