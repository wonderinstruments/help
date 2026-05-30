.PHONY: build build-ui run index search dev-ui test clean install

build:
	go build -o bin/help ./cmd/help

build-ui:
	cd ui/frontend && npm install && npm run build
	go build -o bin/help-ui ./ui/

run:
	go run ./cmd/help

index:
	go run ./cmd/help index

search:
	go run ./cmd/help search $(QUERY)

dev-ui:
	cd ui && wails dev

test:
	go test ./...

install: build build-ui
	cp bin/help ~/.local/bin/help
	cp bin/help-ui ~/.local/bin/help-ui

clean:
	rm -rf bin/ ui/build/ ui/frontend/dist/
	rm -f ~/.cache/help/index.db
