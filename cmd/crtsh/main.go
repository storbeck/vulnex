package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
)

type CertificateEntry struct {
	CommonName string `json:"common_name"`
}

func queryCrtSh(domain string) ([]string, error) {
	// Construct the URL
	url := fmt.Sprintf("https://crt.sh/?q=%s&output=json", domain)

	// Create HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
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
		fmt.Println("Usage: crtsh <domain>")
		os.Exit(1)
	}

	domain := os.Args[1]

	// Query crt.sh
	results, err := queryCrtSh(domain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	for _, name := range results {
		fmt.Println(name)
	}
}