package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Connect to SQLite database
	db, err := sql.Open("sqlite3", "vulnex.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize table
	_, err = db.Exec(`DROP TABLE IF EXISTS files`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT,
		path TEXT,
		status_code INTEGER,
		content_length INTEGER,
		discovered_at TIMESTAMP,
		UNIQUE(url, path)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Read target URLs from stdin
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter URLs (one per line), press Ctrl+D when done:")
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	// Wordlist path
	wordlist := os.Getenv("WORDLIST_PATH")
	if wordlist == "" {
		log.Fatal("WORDLIST_PATH environment variable not set. Please set it to your wordlist file path.")
	}


	for _, url := range urls {
		fmt.Printf("Scanning %s...\n", url)

		// Execute ffuf with JSON output
		ffufCmd := exec.Command("ffuf",
			"-u", url+"/FUZZ",
			"-w", wordlist,
			"-o", "output.json",
			"-of", "json",
			"-mc", "200,301,403") // Match specific status codes

		if err := ffufCmd.Run(); err != nil {
			log.Printf("Error running ffuf: %v\n", err)
			continue
		}

		// Parse ffuf JSON output
		file, err := os.Open("output.json")
		if err != nil {
			log.Printf("Error opening ffuf output: %v\n", err)
			continue
		}

		defer file.Close()
		parser := bufio.NewScanner(file)
		for parser.Scan() {
			line := parser.Text()
			if strings.Contains(line, "input") {
				// Simplified parsing for JSON output
				parts := strings.Split(line, ",")
				if len(parts) > 0 {
					path := extractField(parts, `"input"`)
					statusCode := extractField(parts, `"status"`)
					contentLength := extractField(parts, `"length"`)

					// Insert into SQLite
					_, err = db.Exec(`INSERT OR IGNORE INTO files 
					(url, path, status_code, content_length, discovered_at)
					VALUES (?, ?, ?, ?, ?)`,
						url, path, statusCode, contentLength, time.Now())

					if err != nil {
						log.Printf("Error inserting record: %v\n", err)
					} else {
						fmt.Printf("[+] Found: %s [%s]\n", path, statusCode)
					}
				}
			}
		}
	}
}

// Helper to extract JSON field values from ffuf output
func extractField(parts []string, field string) string {
	for _, part := range parts {
		if strings.Contains(part, field) {
			return strings.Split(part, ":")[1]
		}
	}
	return ""
}
