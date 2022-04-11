.PHONY: build
build: fmt vet ## Build binary.
	go build -o bin/r53-migrate main.go

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: clean
clean:
	rm -rf bin/
	rm ./*.json