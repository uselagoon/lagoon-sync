pre-build:
	./cmd/addAssets.sh

build:
	CGO_ENABLED=0 go build -o ./builds/lagoon-sync main.go

local-snapshot:
	goreleaser release --skip-publish --skip-sign --rm-dist

clean:
	rm -rf dist/

temp-remove-rsync-binary:
	rm /usr/bin/rsync