# Vulnex - Domain Enumeration Toolkit

A set of tools for domain enumeration and reconnaissance that stores results in SQLite.

## Tools

### discover-domains
Find domains from SSL certificates:
```bash
echo "example.com" | ./bin/discover-domains
```

### discover-subs
Enumerate subdomains:
```bash
echo "example.com" | ./bin/discover-subs
```

### discover-web
Probe domains for web server information:
```bash
cat domains.txt | ./bin/discover-web
```

### discover-urls
Extract URLs from web pages:
```bash
cat domains.txt | ./bin/discover-urls
```

### discover-endpoints
Crawl and discover endpoints using katana:
```bash
cat urls.txt | ./bin/discover-endpoints
```

### discover-files
Find sensitive files and directories:
```bash
cat urls.txt | ./bin/discover-files
```

## Building

Build all tools:
```bash
make all
```

Clean build artifacts:
```bash
make clean
```

## Database

Results are stored in `vulnex.db`. Here are some useful queries:

View all found domains from certificates:
```bash
sqlite3 vulnex.db "SELECT domain,source FROM domains WHERE source='cert'"
```

View all subdomains:
```bash
sqlite3 vulnex.db "SELECT domain FROM domains"
```

View web server details:
```bash
sqlite3 vulnex.db "SELECT domain,status_code,title,tech,server FROM assets"
```

View domains with specific technology:
```bash
sqlite3 vulnex.db "SELECT domain,tech FROM assets WHERE tech LIKE '%nginx%'"
```

View all live domains with status 200:
```bash
sqlite3 vulnex.db "SELECT domain,status_code,server FROM assets WHERE status_code=200"
```

View discovered files and directories:
```bash
sqlite3 vulnex.db "SELECT url,status_code,content_type,content_length FROM files WHERE status_code=200"
```

Reset the database:
```bash
make clean-db
```

## Requirements

- Go 1.23+
- httpx (`go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest`)
- subfinder
- katana
- sqlite3

## Pipeline Example

Run full enumeration pipeline:
```bash
./bin/discover-domains "domain.com" | tee domains.txt
cat domains.txt | ./bin/discover-subs | tee -a domains.txt
cat domains.txt | sort -u | ./bin/discover-web | tee web.txt
cat web.txt | ./bin/discover-urls | tee urls.txt
cat urls.txt | ./bin/discover-endpoints | tee endpoints.txt
cat urls.txt | ./bin/discover-files | tee files.txt
```

## Report

Run `./run.sh` to generate a report in `report.md`.
