.DEFAULT_GOAL := all
all: gomodtidy update-licenses fieldalignment fmt

test:
	go test -v -race ./...

fmt:
	go run golang.org/x/tools/cmd/goimports@v0.21.0 -w .

gomodtidy:
	go mod tidy -v

fieldalignment:
	go run golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@v0.21.0 -test=false $(shell go list ./... | grep -v test)

update-licenses:
	go run github.com/elastic/go-licenser@v0.4.1 -ext .go .
