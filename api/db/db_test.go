package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnection(t *testing.T) {
	// Skip this test if PostgreSQL is not available
	t.Skip("Skipping database connection test. Run this test manually when PostgreSQL is available.")

	// Test connection string
	dbURL := "postgresql://zetafast:zetafast@localhost:5432/zetafast?sslmode=disable"

	// Try to connect to the database
	database, err := NewDB(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Test if we can ping the database
	err = database.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Test creating a table
	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255),
			created_at TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test inserting data
	_, err = database.Exec(`
		INSERT INTO test_table (name, created_at)
		VALUES ($1, $2)
	`, "test_name", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test querying data
	var name string
	var createdAt time.Time
	err = database.QueryRow(`
		SELECT name, created_at
		FROM test_table
		WHERE name = $1
	`, "test_name").Scan(&name, &createdAt)
	if err != nil {
		t.Fatalf("Failed to query test data: %v", err)
	}

	// Verify the data
	assert.Equal(t, "test_name", name)

	// Clean up
	_, err = database.Exec(`DROP TABLE test_table`)
	if err != nil {
		t.Fatalf("Failed to drop test table: %v", err)
	}
}
