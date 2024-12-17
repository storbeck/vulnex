# Vulnex

Vulnex is a fast domain discovery and asset mapping tool that processes HackerOne program scopes and discovers related domains, endpoints, and web assets.

## Features

- 🔍 Certificate transparency log enumeration
- 🌐 Automated subdomain discovery
- 🚦 Web technology fingerprinting
- 📍 Endpoint crawling and parameter discovery
- 📊 Asset reporting and analysis
- 💾 Persistent SQLite storage

## Install

```bash
# Clone the repo
git clone https://github.com/storbeck/vulnex
cd vulnex

# Install dependencies
go mod download
playwright-go install

# Build
make
```

## Usage

```bash
# Basic usage - provide a HackerOne program name
./run.sh <hackerone-program-name>

# Example - scan roblox's HackerOne program scope
./run.sh roblox
```

## Output

Results are stored in `scans/YYYYMMDD_HHMMSS_program/`:
- `domains.txt` - Discovered domains from program scope
- `web.txt` - Active web servers
- `endpoints.txt` - Discovered endpoints
- `files.txt` - Found assets
- `report.md` - Summary report

## Requirements

- Go 1.19+
- SQLite3
- subfinder
- httpx
- katana

## Notes

- Results are stored in SQLite database (`vulnex.db`)
- Use `make clean` to remove binaries
- Use `make clean-db` to remove database

## License

MIT
