BINARY := spektacular
VERSION := 0.1.0

.PHONY: build test lint clean install cross

build:
	go build -ldflags "-X github.com/nicholasjackson/spektacular/cmd.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY)-*

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)

cross:
	GOOS=darwin  GOARCH=arm64 go build -o $(BINARY)-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64 go build -o $(BINARY)-darwin-amd64  .
	GOOS=linux   GOARCH=amd64 go build -o $(BINARY)-linux-amd64   .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY)-windows-amd64.exe .
