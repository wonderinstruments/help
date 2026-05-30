.PHONY: build build-ui run index search dev-ui test clean

build:
	go build -o bin/help ./cmd/help

build-ui:
	cd ui && wails build

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

clean:
	rm -rf bin/ ui/build/
	rm -f ~/.cache/help/index.db
