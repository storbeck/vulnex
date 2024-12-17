package main

import (
    "bufio"
    "bytes"
    "database/sql"
    "fmt"
    "log"
    "os"
    "os/exec"
    "strings"

    _ "github.com/mattn/go-sqlite3"
)

func main() {
    // Initialize database
    db, err := sql.Open("sqlite3", "vulnex.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create table silently
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS domains (
        domain TEXT PRIMARY KEY,
        source TEXT
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Create a temporary file for domain list
    tmpFile, err := os.CreateTemp("", "domains-*.txt")
    if err != nil {
        log.Fatal("Error creating temporary file:", err)
    }
    defer os.Remove(tmpFile.Name())

    // Write domains to temporary file
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        domain := scanner.Text()
        if domain == "" {
            continue
        }
        fmt.Fprintln(tmpFile, domain)
    }
    tmpFile.Close()

    // Run subfinder once with domain list
    cmd := exec.Command("subfinder", "-dL", tmpFile.Name())
    var out bytes.Buffer
    cmd.Stdout = &out
    err = cmd.Run()
    if err != nil {
        log.Fatal("Error running subfinder:", err)
    }

    // Store and output results
    subdomains := strings.Split(out.String(), "\n")
    for _, subdomain := range subdomains {
        subdomain = strings.TrimSpace(subdomain)
        if subdomain == "" {
            continue
        }
        
        // Store in database silently
        _, err := db.Exec("INSERT OR IGNORE INTO domains (domain, source) VALUES (?, ?)", 
            subdomain, "subfinder")
        if err != nil {
            log.Printf("Error storing domain %s: %v\n", subdomain, err)
        }

        fmt.Println(subdomain)
    }
}