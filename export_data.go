package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq" // Ensure you have the correct driver for your database
)

// Record represents a single training example for the AI
type Record struct {
	Text  string `json:"text"`
	Label int    `json:"label"` // The IPC Category mapped to an integer (0 to 7)
}

func main() {
	// 1. Connect to the Argos database
	// Update this connection string with your actual credentials
	connStr := "user=your_user dbname=argos sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %v", err))
	}
	defer db.Close()

	// 2. Create the JSONL file
	file, err := os.Create("argos_dataset.jsonl")
	if err != nil {
		panic(fmt.Errorf("failed to create file: %v", err))
	}
	defer file.Close()

	// 3. Query the data
	// IMPORTANT: You must map the IPC sections (A, B, C...) to integers.
	// You can do this in the SQL query or handle it here in Go.
	query := `
		SELECT abstract, category_id 
		FROM patents 
		WHERE abstract IS NOT NULL
	`
	rows, err := db.Query(query)
	if err != nil {
		panic(fmt.Errorf("failed to execute query: %v", err))
	}
	defer rows.Close()

	encoder := json.NewEncoder(file)
	count := 0

	// 4. Stream rows and encode as JSON lines
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.Text, &record.Label); err != nil {
			fmt.Printf("Warning: Skipping row due to scan error: %v\n", err)
			continue
		}

		if err := encoder.Encode(record); err != nil {
			fmt.Printf("Warning: Skipping row due to JSON encoding error: %v\n", err)
			continue
		}
		count++
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error iterating rows: %v\n", err)
	}

	fmt.Printf("✅ Export complete! %d patents saved to 'argos_dataset.jsonl'.\n", count)
}