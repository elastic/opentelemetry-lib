.DEFAULT_GOAL := all
all: gomodtidy update-licenses fieldalignment fmt

test:
	go test -v -race ./...

fmt:
	go run -modfile=tools/go.mod golang.org/x/tools/cmd/goimports -w .

gomodtidy:
	go mod tidy -v

fieldalignment:
	go run -modfile=tools/go.mod golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment -test=false $(shell go list ./... | grep -v test)

update-licenses:
	go run -modfile=tools/go.mod github.com/elastic/go-licenser -ext .go .
