package main

import (
    "bufio"
    "database/sql"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
    "crypto/sha256"
    _ "github.com/mattn/go-sqlite3"
)

var (
    commonExtensions = []string{
        ".html", ".php", ".asp", ".aspx", ".jsp", ".js", ".css", ".json", 
        ".xml", ".txt", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".zip", 
        ".bak", ".backup", ".swp", ".old", ".env", ".conf", ".config",
        ".sql", ".db", ".sqlite", ".yml", ".yaml", ".log", ".htaccess",
        ".git", ".svn", ".hg", ".DS_Store", "robots.txt", "sitemap.xml",
    }

    commonPaths = []string{
        "admin", "login", "wp-admin", "administrator", "backend",
        "api", "v1", "v2", "v3", "test", "dev", "development",
        "staging", "prod", "production", "backup", "backups",
        "wp-content", "wp-includes", "upload", "uploads", "files",
        "images", "img", "css", "js", "javascript", "assets",
        "static", "media", "docs", "documentation", ".git",
        ".env", "config", "configuration", "setup", "install",
        "phpinfo.php", "info.php", ".htaccess", "web.config",
    }
)

type Response struct {
    URL          string
    StatusCode   int
    ContentType  string
    ContentSize  int64
    BodyHash     string
    Location     string // For redirects
}

func main() {
    db, err := sql.Open("sqlite3", "vulnex.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Drop and recreate the table to ensure clean schema
    _, err = db.Exec(`DROP TABLE IF EXISTS files`)
    if err != nil {
        log.Fatal("Error dropping table:", err)
    }

    // Create files table with all needed columns
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT,
        path TEXT,
        status_code INTEGER,
        content_type TEXT,
        content_length INTEGER,
        body_hash TEXT,
        location TEXT,
        discovered_at TIMESTAMP,
        UNIQUE(url, path)
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Read URLs from stdin
    var urls []string
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        urls = append(urls, scanner.Text())
    }

    // Create work channel and wait group
    jobs := make(chan string)
    var wg sync.WaitGroup

    // Start worker goroutines
    for i := 0; i < 10; // Adjust concurrency level
        i++ {
        wg.Add(1)
        go worker(jobs, &wg, db)
    }

    // Send jobs to workers
    for _, baseURL := range urls {
        // Generate paths to check
        for _, path := range commonPaths {
            jobs <- fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), path)
            
            // Also check with common extensions
            for _, ext := range commonExtensions {
                jobs <- fmt.Sprintf("%s/%s%s", strings.TrimRight(baseURL, "/"), path, ext)
            }
        }
    }

    // Close jobs channel and wait for workers to finish
    close(jobs)
    wg.Wait()
}

func worker(jobs <-chan string, wg *sync.WaitGroup, db *sql.DB) {
    defer wg.Done()

    // Keep track of response hashes per domain to detect duplicates
    domainHashes := make(map[string]string)

    client := &http.Client{
        Timeout: 10 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }

    for urlStr := range jobs {
        parsedURL, err := url.Parse(urlStr)
        if err != nil {
            continue
        }

        resp, err := client.Get(urlStr)
        if err != nil {
            continue
        }

        // Read response body
        body, err := io.ReadAll(io.LimitReader(resp.Body, 8192)) // Limit to first 8KB
        resp.Body.Close()
        if err != nil {
            continue
        }

        // Calculate body hash
        hash := fmt.Sprintf("%x", sha256.Sum256(body))
        
        // Get domain for hash comparison
        domain := parsedURL.Host

        // Check if this is a unique response
        isInteresting := false
        switch resp.StatusCode {
        case http.StatusOK:
            // Check if response is unique for this domain
            if lastHash, exists := domainHashes[domain]; !exists || lastHash != hash {
                isInteresting = true
                // Only store hash if response size is significant
                if len(body) > 100 { // Adjust threshold as needed
                    domainHashes[domain] = hash
                }
            }
        case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect:
            // Include redirects to interesting locations
            location := resp.Header.Get("Location")
            if strings.Contains(location, "login") || 
               strings.Contains(location, "admin") ||
               strings.Contains(location, "dashboard") {
                isInteresting = true
            }
        case http.StatusUnauthorized, http.StatusForbidden:
            // Authentication/Authorization endpoints are interesting
            isInteresting = true
        case http.StatusNotFound:
            // Ignore 404s
            continue
        default:
            // Other status codes might be interesting
            isInteresting = true
        }

        if isInteresting {
            // Store in database with matching column name
            _, err = db.Exec(`
                INSERT OR IGNORE INTO files 
                (url, path, status_code, content_type, content_length, body_hash, location, discovered_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
                urlStr,
                parsedURL.Path,
                resp.StatusCode,
                resp.Header.Get("Content-Type"),
                len(body),
                hash,
                resp.Header.Get("Location"),
            )
            if err != nil {
                log.Printf("Error storing file %s: %v\n", urlStr, err)
            }

            // Also filter out zero-length responses
            if len(body) > 0 {
                switch resp.StatusCode {
                case http.StatusOK:
                    fmt.Printf("[200] %s (%d bytes)\n", urlStr, len(body))
                case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect:
                    fmt.Printf("[%d] %s -> %s\n", resp.StatusCode, urlStr, resp.Header.Get("Location"))
                default:
                    fmt.Printf("[%d] %s (%s)\n", resp.StatusCode, urlStr, resp.Header.Get("Content-Type"))
                }
            }
        }
    }
}