BINARY := bin/policystack-validator
PKG := ./cmd/policystack-validator

.PHONY: build test lint fmt clean run tidy

build:
	go build -trimpath -o $(BINARY) $(PKG)

test:
	go test ./internal/...

lint:
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt needed (run 'make fmt'):"; echo "$$unformatted"; exit 1; fi
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -rf bin

run: build
	./$(BINARY) --repo-root ../..
