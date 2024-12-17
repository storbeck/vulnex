package main

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "sort"
    "strings"
)

type CertificateEntry struct {
    CommonName string `json:"common_name"`
}

func queryCrtSh(domain string) ([]string, error) {
    // URL encode the domain and construct the URL properly
    url := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", domain)

    // Add User-Agent header to avoid being blocked
    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; vulnex/1.0)")

    // Make the request
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to make request: %v", err)
    }
    defer resp.Body.Close()

    // Read and check response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %v", err)
    }

    // Add debug output
    if !strings.Contains(string(body), "[{") {
        return nil, fmt.Errorf("unexpected response from crt.sh: %s", string(body)[:100])
    }

    // Parse JSON
    var entries []CertificateEntry
    err = json.Unmarshal(body, &entries)
    if err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %v", err)
    }

    // Extract unique common names
    uniqueNames := make(map[string]bool)
    for _, entry := range entries {
        if entry.CommonName != "" {
            uniqueNames[entry.CommonName] = true
        }
    }

    // Convert to sorted slice
    names := make([]string, 0, len(uniqueNames))
    for name := range uniqueNames {
        names = append(names, name)
    }
    sort.Strings(names)

    return names, nil
}

func main() {
    // Check if domain is provided
    if len(os.Args) < 2 {
        fmt.Println("Usage: enum-cert <domain>")
        os.Exit(1)
    }

    domain := os.Args[1]

    // Open database connection
    db, err := sql.Open("sqlite3", "vulnex.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create domains table if it doesn't exist
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS domains (
        domain TEXT PRIMARY KEY,
        source TEXT,
        discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Query crt.sh
    results, err := queryCrtSh(domain)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Store and print results
    for _, name := range results {
        // Insert domain into database
        _, err := db.Exec(`
            INSERT OR IGNORE INTO domains (domain, source)
            VALUES (?, ?)`,
            name, "crt.sh",
        )
        if err != nil {
            log.Printf("Error storing domain %s: %v\n", name, err)
        }
        fmt.Println(name)
    }
}