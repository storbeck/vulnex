.PHONY: all clean install clean-db

all:
	go build -o bin/enum-cert cmd/crtsh/main.go
	go build -o bin/enum-sub cmd/subfinder/main.go
	go build -o bin/enum-web cmd/httpx/main.go

clean:
	rm -rf bin

clean-db:
	rm -f assets.db vulnex.db

install:
	go install ./cmd/crtsh
	go install ./cmd/subfinder
	go install ./cmd/httpx