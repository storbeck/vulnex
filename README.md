# Vulnex - Domain Enumeration Toolkit

A set of tools for domain enumeration and reconnaissance that stores results in SQLite.

## Tools

### enum-cert
Find domains from SSL certificates:
```bash
echo "example.com" | ./bin/enum-cert
```

### enum-sub
Enumerate subdomains:
```bash
echo "example.com" | ./bin/enum-sub
```

### enum-web
Probe domains for web server information:
```bash
cat domains.txt | ./bin/enum-web
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

Reset the database:
```bash
make clean-db
```

## Requirements

- Go 1.23+
- httpx (`go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest`)
- subfinder
- sqlite3

## Pipeline Example

Run full enumeration pipeline:
```bash
echo "example.com" | ./bin/enum-cert | tee domains.txt
cat domains.txt | ./bin/enum-sub | tee -a domains.txt
cat domains.txt | sort -u | ./bin/enum-web
```