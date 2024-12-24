package main

import (
	"context"
	"fmt"

	pbloom "github.com/godwhoa/pbloom/go"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
)

// File represents a database row
type File struct {
	File   string `db:"file"`
	Field  string `db:"field"`
	Filter []byte `db:"filter"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Sample data per file
	data := map[string]map[string][]string{
		"file1.parquet": {
			"name":  {"John", "Doe", "Alice", "Bob"},
			"email": {"john@acme.com", "doe@acme.com", "alice@acme.com"},
			"date":  {"2020-01-01", "2020-01-02"},
		},
		"file2.parquet": {
			"name":  {"Jane", "Doe"},
			"email": {"jane@acme.com", "doe@acme.com"},
			"date":  {"2020-01-03", "2020-01-01"},
		},
	}

	// Desired false positive rate
	fpRate := 0.01

	// Initialize a map to hold Bloom filters for each file and field
	fileFilters := make(map[string]map[string]*pbloom.Filter)

	for file, fields := range data {
		fileFilters[file] = make(map[string]*pbloom.Filter)
		for field, values := range fields {
			// Extract unique values using lo.Uniq
			uniqueValues := lo.Uniq(values)
			count := len(uniqueValues)

			// Create a new Bloom filter with the number of unique entries and desired false positive rate
			filter, err := pbloom.NewFilterFromEntriesAndFP(count, fpRate)
			must(err)

			// Add each unique value to the Bloom filter
			for _, value := range uniqueValues {
				filter.Put([]byte(value))
			}

			fileFilters[file][field] = filter
		}
	}

	// Serialize the Bloom filters for storage
	serializedFilters := make([]File, 0)

	for file, fields := range fileFilters {
		for field, filter := range fields {
			serialized, err := filter.Serialize()
			must(err)
			serializedFilters = append(serializedFilters, File{
				File:   file,
				Field:  field,
				Filter: serialized,
			})
		}
	}

	// Connect to the PostgreSQL database using sqlx
	db, err := sqlx.Connect("pgx", "postgres://user:password@localhost:5432/test")
	must(err)
	defer db.Close()

	// Create the filters table if it doesn't exist
	_, err = db.ExecContext(context.TODO(), `
		CREATE TABLE IF NOT EXISTS filters (
			file TEXT,
			field TEXT,
			filter BYTEA,
			PRIMARY KEY (file, field)
		);
		CREATE EXTENSION IF NOT EXISTS pbloompg;
		TRUNCATE TABLE filters;
	`)
	must(err)

	// Insert or update Bloom filters in the database
	for _, f := range serializedFilters {
		_, err = db.NamedExecContext(context.TODO(),
			`INSERT INTO filters (file, field, filter)
			 VALUES (:file, :field, :filter)`,
			f)
		must(err)
	}

	// Example queries
	queryBloomFilter(db, "name", "John")
	queryBloomFilter(db, "date", "2020-01-01")
}

// queryBloomFilter checks which files contain a given value for a specified field using Bloom filters.
func queryBloomFilter(db *sqlx.DB, field string, value string) {
	var files []string
	err := db.SelectContext(context.TODO(), &files,
		`SELECT file FROM filters
		 WHERE field = $1 AND pbloom_contains(filter, $2)`,
		field, []byte(value))
	must(err)
	fmt.Printf("Files with rows %s=%s: %v\n", field, value, files)
}
