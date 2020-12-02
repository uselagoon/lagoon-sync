build:
	CGO_ENABLED=0 go build -o ./builds/lagoon-sync main.go

local-snapshot:
	goreleaser --snapshot --skip-publish --rm-dist
