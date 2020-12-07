DIR := $(PWD)
GOCMD=go
GOPATH:=$(shell $(GOCMD) env GOPATH 2> /dev/null)
GOOS:=$(shell $(GOCMD) env GOOS 2> /dev/null)
GOARCH:=$(shell $(GOCMD) env GOARCH 2> /dev/null)

VERSION=$(shell git describe --tags --abbrev=0)
VERSION_FORMATTED=$(shell git describe --tags --abbrev=0 | sed 's/\./-/g')

# Prep
check-go:
ifndef GOPATH
	$(error "go is not available, please install")
endif

pre-build: check-go
	go-embed -compress=false -input binaries/ -output assets/main.go

# Build
local-build: pre-build
	GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOCMD) build -o builds/lagoon-sync-${VERSION_FORMATTED}-${GOOS} -v

local-build-linux: pre-build
		GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOCMD) build -o builds/lagoon-sync-${VERSION_FORMATTED}-linux -v

local-build-darwin: pre-build
		GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOCMD) build -o builds/lagoon-sync-${VERSION_FORMATTED}-darwin -v

# Release
release-test: pre-build
	goreleaser release --skip-publish --skip-sign --rm-dist

#https://github.com/fmahnke/shell-semver
release-patch: release-test
	$(eval VERSION=$(shell ${PWD}/increment_ver.sh -p $(shell git describe --abbrev=0 --tags)))
	$(VERSION)
	# git tag $(VERSION)
	# git push $(GIT_ORIGIN) main --tags

release-minor: release-test
	$(eval VERSION=$(shell ${PWD}/increment_ver.sh -m $(shell git describe --abbrev=0 --tags)))
	$(VERSION)
	# git tag $(VERSION)
	# git push $(GIT_ORIGIN) main --tags

release-major: release-test
	$(eval VERSION=$(shell ${PWD}/increment_ver.sh -M $(shell git describe --abbrev=0 --tags)))
	$(VERSION)
	# git tag $(VERSION)
	# git push $(GIT_ORIGIN) main --tags

# Clean
clean:
	$(GOCMD) clean
	rm -rf dist/
