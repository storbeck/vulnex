.PHONY: all clean install clean-db

all:
	go build -o bin/crtsh cmd/crtsh/main.go
	go build -o bin/subfinder cmd/subfinder/main.go
	go build -o bin/httpx cmd/httpx/main.go

clean:
	rm -rf bin

clean-db:
	rm -f vulnex.db

install:
	go install ./cmd/crtsh
	go install ./cmd/subfinder
	go install ./cmd/httpx