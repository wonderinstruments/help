default:
    @just --list

build:
    go build -tags fts5 -o bin/help-cli ./cmd/help

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

create-release version: build build-ui
    gh release create "v{{version}}" bin/help-cli bin/help-ui \
        --title "v{{version}}" \
        --notes "Release v{{version}}"
    @echo "Release v{{version}} created. Run 'just update-hashes {{version}}' to update flake.nix"

update-hashes version:
    #!/usr/bin/env bash
    set -euo pipefail
    cli_hash=$(nix-prefetch-url "https://github.com/wonderinstruments/help/releases/download/v{{version}}/help-cli" 2>/dev/null)
    ui_hash=$(nix-prefetch-url "https://github.com/wonderinstruments/help/releases/download/v{{version}}/help-ui" 2>/dev/null)
    sed -i 's|version = ".*";|version = "{{version}}";|' flake.nix
    sed -i "0,/sha256 = \".*\";/s|sha256 = \".*\";|sha256 = \"$cli_hash\";|" flake.nix
    sed -i "$(grep -n 'sha256 = ' flake.nix | tail -1 | cut -d: -f1)s|sha256 = \".*\";|sha256 = \"$ui_hash\";|" flake.nix
    echo "Updated flake.nix to v{{version}}"
    echo "  help-cli: $cli_hash"
    echo "  help-ui:  $ui_hash"

clean:
    rm -rf bin/ ui/build/ ui/frontend/dist/
    rm -f ~/.cache/help/index.db
