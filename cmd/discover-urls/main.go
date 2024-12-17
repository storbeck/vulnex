package main

import (
    "bufio"
    "bytes"
    "database/sql"
    "fmt"
    "log"
    "net/url"
    "os"
    "os/exec"
    "strings"

    _ "github.com/mattn/go-sqlite3"
)

func main() {
    db, err := sql.Open("sqlite3", "vulnex.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create urls table
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
        url TEXT PRIMARY KEY,
        domain TEXT,
        path TEXT,
        params TEXT
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Collect domains from stdin
    var domains []string
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        domain := strings.TrimSpace(scanner.Text())
        if domain != "" {
            domains = append(domains, domain)
        }
    }

    if len(domains) == 0 {
        log.Fatal("No domains provided")
    }

    // Add debug logging
    fmt.Fprintf(os.Stderr, "Processing %d domains...\n", len(domains))

    // Run gau with timeout context
    args := append([]string{
        "--blacklist", "png,jpg,gif,jpeg,css,js,woff,woff2,svg",
        "--threads", "10",
    }, domains...)  // Add domains as command line arguments
    
    cmd := exec.Command("gau", args...)

    // Add error output capture
    var stderr bytes.Buffer
    cmd.Stderr = &stderr

    var out bytes.Buffer
    cmd.Stdout = &out

    // Start the command
    if err := cmd.Start(); err != nil {
        log.Fatal("Error starting gau:", err)
    }

    // Wait for command to complete
    if err := cmd.Wait(); err != nil {
        log.Printf("gau stderr output: %s\n", stderr.String())
        log.Fatal("Error running gau:", err)
    }

    // Process URLs
    scanner = bufio.NewScanner(strings.NewReader(out.String()))
    urlCount := 0
    for scanner.Scan() {
        urlStr := scanner.Text()
        if urlStr == "" {
            continue
        }

        parsedURL, err := url.Parse(urlStr)
        if err != nil {
            log.Printf("Error parsing URL %s: %v\n", urlStr, err)
            continue
        }

        // Store in database
        _, err = db.Exec(`
            INSERT OR IGNORE INTO urls (url, domain, path, params)
            VALUES (?, ?, ?, ?)`,
            urlStr,
            parsedURL.Host,
            parsedURL.Path,
            parsedURL.RawQuery,
        )
        if err != nil {
            log.Printf("Error storing URL %s: %v\n", urlStr, err)
        } else {
            urlCount++
            if urlCount%1000 == 0 {
                fmt.Fprintf(os.Stderr, "Processed %d URLs...\n", urlCount)
            }
            fmt.Println(urlStr)
        }
    }

    fmt.Fprintf(os.Stderr, "Finished processing %d URLs\n", urlCount)
}