.PHONY: build install version release fmt vet test

# Build a local binary (reports a -dirty dev version).
build:
	go build -o gitle .

# Install into your Go bin.
install:
	go install .

version: build
	./gitle --version

fmt:
	gofmt -w .

vet:
	go vet ./...

# Cut a release. Pass BUMP=patch|minor|major or an explicit BUMP=v1.2.3.
#   make release BUMP=patch
release:
	@test -n "$(BUMP)" || { echo "usage: make release BUMP=patch|minor|major|vX.Y.Z"; exit 1; }
	@scripts/release.sh "$(BUMP)"
