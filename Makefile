.PHONY: all clean clean-db


all:
	go build -o bin/discover-domains cmd/discover-domains/main.go
	go build -o bin/discover-subs cmd/discover-subs/main.go
	go build -o bin/discover-web cmd/discover-web/main.go
	go build -o bin/discover-endpoints cmd/discover-endpoints/main.go
	go build -o bin/discover-files cmd/discover-files/main.go

clean:
	rm -rf bin

clean-db:
	rm -f assets.db vulnex.db

