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
    "net/url"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    db, err := sql.Open("sqlite3", "vulnex.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create endpoints table
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS endpoints (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT,
        endpoint TEXT,
        method TEXT,
        params TEXT,
        source TEXT,
        UNIQUE(url, endpoint, method)
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Create temporary file for URLs
    tmpFile, err := os.CreateTemp("", "urls-*.txt")
    if err != nil {
        log.Fatal("Error creating temporary file:", err)
    }
    defer os.Remove(tmpFile.Name())

    // Read URLs from stdin
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        fmt.Fprintln(tmpFile, scanner.Text())
    }
    tmpFile.Close()

    // Run katana
    cmd := exec.Command("katana", 
        "-list", tmpFile.Name(),
        "-jc",              // Enable JavaScript crawling
        "-aff",            // Automatic form fill (previously -automatic-form)
        "-silent",         // Reduce noise
    )
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err = cmd.Run()
    if err != nil {
        log.Printf("Command failed with error: %v", err)
        log.Printf("Stderr output: %s", stderr.String())
        log.Printf("Stdout output: %s", stdout.String())
        log.Fatal("Katana execution failed")
    }

    // Process results
    scanner = bufio.NewScanner(strings.NewReader(stdout.String()))
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
            INSERT OR IGNORE INTO endpoints (url, endpoint, method, params, source)
            VALUES (?, ?, ?, ?, ?)`,
            urlStr,
            parsedURL.Path,
            guessHTTPMethod(parsedURL.Path),
            parsedURL.RawQuery,
            "katana",
        )
        if err != nil {
            log.Printf("Error storing endpoint %s: %v\n", urlStr, err)
        } else {
            fmt.Printf("%s %s\n", guessHTTPMethod(parsedURL.Path), urlStr)
        }
    }
}

func guessHTTPMethod(path string) string {
    path = strings.ToLower(path)
    if strings.Contains(path, "delete") {
        return "DELETE"
    } else if strings.Contains(path, "update") || strings.Contains(path, "edit") {
        return "PUT"
    } else if strings.Contains(path, "create") || strings.Contains(path, "add") {
        return "POST"
    }
    return "GET"
}