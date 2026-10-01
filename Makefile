.PHONY: build test lint coverage clean release-binaries

BINARY := bin/entrypoint
MAIN   := cmd/entrypoint/main.go

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

build:
	CGO_ENABLED=0 go build -ldflags='-w -s' -o $(BINARY) $(MAIN)

test:
	go test ./... -race

lint:
	go vet ./...
	golangci-lint run

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -rf bin/ coverage.out

release-binaries:
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "Building $$os/$$arch..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags='-w -s' \
			-o bin/entrypoint-$$os-$$arch $(MAIN); \
	done
