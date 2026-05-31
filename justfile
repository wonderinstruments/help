default:
    @just --list

build:
    go build -tags fts5 -o bin/help ./cmd/help

build-ui:
    cd ui && wails build
    cp ui/build/bin/help-ui bin/help-ui

run *args:
    go run -tags fts5 ./cmd/help {{args}}

index:
    go run -tags fts5 ./cmd/help index

search query:
    go run -tags fts5 ./cmd/help search "{{query}}"

dev-ui:
    cd ui && wails dev

test:
    go test -tags fts5 ./...

install: build build-ui
    cp bin/help ~/.local/bin/help
    cp bin/help-ui ~/.local/bin/help-ui

clean:
    rm -rf bin/ ui/build/ ui/frontend/dist/
    rm -f ~/.cache/help/index.db
