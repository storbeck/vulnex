package main

import (
    "bufio"
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
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

    // Create assets table
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS assets (
    domain TEXT PRIMARY KEY,
    status_code INTEGER,
    title TEXT,
    tech TEXT,
    server TEXT,
    content_type TEXT,
    tls_version TEXT,
    final_url TEXT,
    favicon_hash TEXT,
    web_server TEXT,
    csp TEXT,
    jarm TEXT,
    cert_chain TEXT,
    ports TEXT
    )`)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }

    // Collect all domains first to run httpx once
    var domains []string
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        domain := strings.TrimSpace(scanner.Text())
        if domain != "" && !strings.HasPrefix(domain, "Database") && !strings.HasPrefix(domain, "Processing") {
            domains = append(domains, domain)
        }
    }

    if len(domains) == 0 {
        log.Fatal("No domains provided")
    }

    // Create temporary file with domains
    tmpfile, err := os.CreateTemp("", "domains")
    if err != nil {
        log.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    for _, domain := range domains {
        fmt.Fprintln(tmpfile, domain)
    }
    tmpfile.Close()

    // Run httpx once with all domains
    cmd := exec.Command("httpx", 
    "-json",
    "-tech-detect", 
    "-title",
    "-server",
    "-tls-grab",
    "-status-code",
    "-content-type",
    "-location",           // Follow redirects and get final URL
    "-favicon",            // Get favicon hash
    "-web-server",         // Web server info
    "-csp-probe",         // Content Security Policy
    "-jarm",              // JARM TLS fingerprint
    "-x509-chain",        // Full certificate chain
    "-ports", "80,443,8080,8443", // Check multiple ports
    "-l", tmpfile.Name())
    var out bytes.Buffer
    cmd.Stdout = &out
    err = cmd.Run()
    if err != nil {
        log.Fatal("Error running httpx:", err)
    }

    // Process each line of JSON output
    scanner = bufio.NewScanner(strings.NewReader(out.String()))
    for scanner.Scan() {
        line := scanner.Text()
        if line == "" {
            continue
        }

        var result map[string]interface{}
        if err := json.Unmarshal([]byte(line), &result); err != nil {
            log.Printf("Error parsing JSON: %v\n", err)
            continue
        }

        // Extract technologies if present
        var tech string
        if technologies, ok := result["technologies"].([]interface{}); ok {
            techStrings := make([]string, len(technologies))
            for i, t := range technologies {
                techStrings[i] = fmt.Sprint(t)
            }
            tech = strings.Join(techStrings, ", ")
        }

        // Extract TLS version if present
		var tlsVersion string
		if tlsData, ok := result["tls"].(map[string]interface{}); ok {
			if version, ok := tlsData["version"].(string); ok {
				tlsVersion = version
			}
		}

		// Store results
		_, err = db.Exec(`
			INSERT OR REPLACE INTO assets
			(domain, status_code, title, tech, server, content_type, tls_version)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			result["url"],
			result["status_code"],
			result["title"],
			tech,
			result["webserver"],
			result["content-type"],
			tlsVersion,  // Use the extracted version instead of the whole map
		)
        if err != nil {
            log.Printf("Error storing asset %v: %v\n", result["url"], err)
        } else {
            fmt.Printf("Stored asset info for: %v\n", result["url"])
        }
    }
}