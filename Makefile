BINARY := acctpass
GO ?= go

.PHONY: build test vet vuln ci clean build-macos build-linux build-windows build-all

build:
	$(GO) build -trimpath -o bin/$(BINARY) .

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: test vet build

clean:
	rm -rf bin

build-macos:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -trimpath -o bin/darwin-arm64/$(BINARY) .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -trimpath -o bin/darwin-amd64/$(BINARY) .

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -trimpath -o bin/linux-amd64/$(BINARY) .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -trimpath -o bin/linux-arm64/$(BINARY) .

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -trimpath -o bin/windows-amd64/$(BINARY).exe .

build-all: build-macos build-linux build-windows
