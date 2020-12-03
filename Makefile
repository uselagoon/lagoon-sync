pre-build:
	go-embed -compress=false -input binaries/ -output assets/main.go

build: pre-build
	CGO_ENABLED=0 go build -o ./builds/lagoon-sync main.go

run: pre-build
	go run main.go

local-snapshot:
	goreleaser release --skip-publish --skip-sign --rm-dist

clean:
	rm -rf dist/