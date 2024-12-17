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

    // Create table - let's make sure this executes
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS domains (
        domain TEXT PRIMARY KEY,
        source TEXT
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Let's test the database is working
    fmt.Println("Database initialized, table created")

    // Read domains from stdin
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        domain := scanner.Text()
        if domain == "" {
            continue
        }

        fmt.Println("Processing domain:", domain)

        // Run subfinder for each domain
        cmd := exec.Command("subfinder", "-d", domain)
        var out bytes.Buffer
        cmd.Stdout = &out
        err = cmd.Run()
        if err != nil {
            log.Printf("Error running subfinder for %s: %v\n", domain, err)
            continue
        }

        // Store results
        subdomains := strings.Split(out.String(), "\n")
        for _, subdomain := range subdomains {
            subdomain = strings.TrimSpace(subdomain)
            if subdomain == "" {
                continue
            }
            
            _, err := db.Exec("INSERT OR IGNORE INTO domains (domain, source) VALUES (?, ?)", 
                subdomain, "subfinder")
            if err != nil {
                log.Printf("Error storing domain %s: %v\n", subdomain, err)
            } else {
                fmt.Println("Stored:", subdomain)
            }
        }
    }
}