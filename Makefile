.PHONY: build build-ui run index search dev-ui test clean install

build:
	go build -tags fts5 -o bin/help ./cmd/help

build-ui:
	cd ui && wails build
	cp ui/build/bin/help-ui bin/help-ui

run:
	go run -tags fts5 ./cmd/help

index:
	go run -tags fts5 ./cmd/help index

search:
	go run -tags fts5 ./cmd/help search $(QUERY)

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
