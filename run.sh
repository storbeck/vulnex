#!/bin/bash

# Check if domain argument is provided
if [ -z "$1" ]; then
    echo "Usage: ./run.sh <domain>"
    exit 1
fi

DOMAIN=$1
echo "[+] Starting enumeration for $DOMAIN"

# Create output directory with timestamp
OUTDIR="scans/$(date +%Y%m%d_%H%M%S)_$DOMAIN"
mkdir -p "$OUTDIR"

echo "[+] Discovering domains..."
./bin/discover-domains "$DOMAIN" | tee "$OUTDIR/domains.txt"

echo "[+] Enumerating subdomains..."
cat "$OUTDIR/domains.txt" | ./bin/discover-subs | tee -a "$OUTDIR/domains.txt"

echo "[+] Sorting unique domains..."
sort -u "$OUTDIR/domains.txt" -o "$OUTDIR/domains.txt"

echo "[+] Probing web servers..."
cat "$OUTDIR/domains.txt" | ./bin/discover-web | tee "$OUTDIR/web.txt"

echo "[+] Extracting URLs..."
cat "$OUTDIR/web.txt" | ./bin/discover-urls | tee "$OUTDIR/urls.txt"

echo "[+] Discovering endpoints..."
cat "$OUTDIR/urls.txt" | ./bin/discover-endpoints | tee "$OUTDIR/endpoints.txt"

echo "[+] Finding sensitive files..."
cat "$OUTDIR/urls.txt" | ./bin/discover-files | tee "$OUTDIR/files.txt"

echo "[+] Generating final report..."
{
    echo "# Security Reconnaissance Report for $DOMAIN"
    echo "**Scan Date:** $(date)"
    echo
    echo "## Overview"
    echo "- Target Domain: $DOMAIN"
    echo "- Total Domains: $(wc -l < "$OUTDIR/domains.txt")"
    echo "- Web Servers: $(wc -l < "$OUTDIR/web.txt")"
    echo "- URLs Found: $(wc -l < "$OUTDIR/urls.txt")"
    echo "- Endpoints Discovered: $(wc -l < "$OUTDIR/endpoints.txt")"
    echo "- Files/Directories Found: $(wc -l < "$OUTDIR/files.txt")"
    echo
    echo "## Discovered Domains"
    echo '```'
    cat "$OUTDIR/domains.txt"
    echo '```'
    echo
    echo "## Web Servers"
    echo '```'
    cat "$OUTDIR/web.txt"
    echo '```'
    echo
    echo "## URLs"
    echo '```'
    cat "$OUTDIR/urls.txt"
    echo '```'
    echo
    echo "## Endpoints"
    echo '```'
    cat "$OUTDIR/endpoints.txt"
    echo '```'
    echo
    echo "## Files and Directories"
    echo '```'
    cat "$OUTDIR/files.txt"
    echo '```'
    echo
    echo "## Database Insights"
    echo "### Technology Distribution"
    echo '```'
    sqlite3 vulnex.db "SELECT tech, COUNT(*) as count FROM assets WHERE tech IS NOT NULL GROUP BY tech ORDER BY count DESC"
    echo '```'
    echo
    echo "### Status Code Distribution"
    echo '```'
    sqlite3 vulnex.db "SELECT status_code, COUNT(*) as count FROM assets GROUP BY status_code ORDER BY count DESC"
    echo '```'
    echo
    echo "### Interesting Files (200 OK)"
    echo '```'
    sqlite3 vulnex.db "SELECT url, content_type, content_length FROM files WHERE status_code=200 ORDER BY content_length DESC LIMIT 20"
    echo '```'
} > "$OUTDIR/report.md"

echo "[+] Scan complete! Results saved in $OUTDIR/"
echo "[+] Found:"
echo "    $(wc -l < "$OUTDIR/domains.txt") domains"
echo "    $(wc -l < "$OUTDIR/web.txt") web servers"
echo "    $(wc -l < "$OUTDIR/urls.txt") URLs"
echo "    $(wc -l < "$OUTDIR/endpoints.txt") endpoints"
echo "    $(wc -l < "$OUTDIR/files.txt") files"
echo "[+] Full report available at: $OUTDIR/report.md"
