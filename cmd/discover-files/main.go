package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/playwright-community/playwright-go"
)

func main() {
	// Connect to SQLite database
	db, err := sql.Open("sqlite3", "vulnex.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize table
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
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	// Initialize Playwright
	err = playwright.Install()
	if err != nil {
		log.Fatalf("Could not install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	// Create a new context
	context, err := browser.NewContext()
	if err != nil {
		log.Fatalf("Could not create context: %v", err)
	}
	defer context.Close()

	// Process each URL
	for _, baseURL := range urls {
		fmt.Printf("Crawling %s...\n", baseURL)

		// Create a new page
		page, err := context.NewPage()
		if err != nil {
			log.Printf("Could not create page: %v", err)
			continue
		}

		// Enable request interception
		page.On("request", func(req playwright.Request) {
			url := req.URL()
			fmt.Printf("Discovered: %s\n", url)

			// Store in database
			_, err = db.Exec(`INSERT OR IGNORE INTO files 
				(url, path, status_code, content_length, discovered_at)
				VALUES (?, ?, ?, ?, ?)`,
				baseURL, req.URL(), 200, 0, time.Now())
			if err != nil {
				log.Printf("Error inserting record: %v\n", err)
			}
		})

		// Navigate to the page
		_, err = page.Goto(baseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(30000),
		})
		if err != nil {
			log.Printf("Error navigating to %s: %v\n", baseURL, err)
			continue
		}

		// Get all resources loaded by the page
		resources, err := page.Evaluate(`() => {
			const resources = [];
			performance.getEntriesByType('resource').forEach(entry => {
				resources.push(entry.name);
			});
			return resources;
		}`)
		if err != nil {
			log.Printf("Error getting resources: %v\n", err)
			continue
		}

		fmt.Printf("Resources found on %s:\n", baseURL)
		for _, resource := range resources.([]interface{}) {
			fmt.Printf("- %s\n", resource)
		}

		page.Close()
	}
}
