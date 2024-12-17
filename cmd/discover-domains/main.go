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
    "encoding/csv"
)

type CertificateEntry struct {
    CommonName string `json:"common_name"`
}

type ScopeEntry struct {
    Identifier string
    AssetType  string
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

    // Check if response is valid JSON array (even if empty)
    bodyStr := string(body)
    if !strings.HasPrefix(bodyStr, "[") {
        bodyPreview := bodyStr
        if len(bodyPreview) > 100 {
            bodyPreview = bodyPreview[:100]
        }
        return nil, fmt.Errorf("unexpected response from crt.sh: %s", bodyPreview)
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

func getScopeFromCSV(program string) (map[string]bool, error) {
    url := fmt.Sprintf("https://hackerone.com/teams/%s/assets/download_csv.csv", program)
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch scope CSV: %v", err)
    }
    defer resp.Body.Close()

    reader := csv.NewReader(resp.Body)
    // Skip header row
    _, err = reader.Read()
    if err != nil {
        return nil, fmt.Errorf("failed to read CSV header: %v", err)
    }

    scopeDomains := make(map[string]bool)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("error reading CSV: %v", err)
        }

        identifier := record[0]
        assetType := record[1]

        // Only include WILDCARD and URL types, skip executables and other asset types
        if assetType != "WILDCARD" && assetType != "URL" {
            continue
        }

        // Remove wildcard prefix if present
        identifier = strings.TrimPrefix(identifier, "*.")
        scopeDomains[identifier] = true
    }

    return scopeDomains, nil
}

func isInScope(domain string, scopeDomains map[string]bool) bool {
    parts := strings.Split(domain, ".")
    for i := 0; i < len(parts); i++ {
        testDomain := strings.Join(parts[i:], ".")
        if scopeDomains[testDomain] {
            return true
        }
    }
    return false
}

func main() {
    // Check if program name is provided
    if len(os.Args) != 2 {
        fmt.Println("Usage: discover-domains <h1-program>")
        os.Exit(1)
    }

    program := os.Args[1]

    // Fetch scope from CSV
    scopeDomains, err := getScopeFromCSV(program)
    if err != nil {
        log.Fatalf("Error fetching scope: %v", err)
    }

    if len(scopeDomains) == 0 {
        log.Fatal("No valid domains found in scope")
    }

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

    // Process each domain in scope
    for domain := range scopeDomains {
        // Query crt.sh for each domain
        results, err := queryCrtSh(domain)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error querying %s: %v\n", domain, err)
            continue
        }

        // Filter and store results
        for _, name := range results {
            if isInScope(name, scopeDomains) {
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
    }
}