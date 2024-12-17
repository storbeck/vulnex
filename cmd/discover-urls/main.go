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

    // Run gau
    cmd := exec.Command("gau",
        "--blacklist", "png,jpg,gif,jpeg,css,js,woff,woff2,svg",
    )

    // Set up pipe to send domains to gau
    stdin, err := cmd.StdinPipe()
    if err != nil {
        log.Fatal(err)
    }

    var out bytes.Buffer
    cmd.Stdout = &out

    // Start the command
    if err := cmd.Start(); err != nil {
        log.Fatal("Error starting gau:", err)
    }

    // Write domains to gau's stdin
    for _, domain := range domains {
        fmt.Fprintln(stdin, domain)
    }
    stdin.Close()

    // Wait for command to complete
    if err := cmd.Wait(); err != nil {
        log.Fatal("Error running gau:", err)
    }

    // Process URLs
    scanner = bufio.NewScanner(strings.NewReader(out.String()))
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
            fmt.Println(urlStr)
        }
    }
}