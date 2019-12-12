.PHONY: build
build:
	go build -v

.PHONY: test
test:
	go test -cover -race ./...

.PHONY: vet
vet:
	go vet ./...
