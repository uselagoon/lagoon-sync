DIR := $(PWD)
GIT_ORIGIN=origin
BUILD := $(shell git rev-parse --short HEAD)
PROJECTNAME := $(shell basename "$(PWD)")

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOPATH:=$(shell $(GOCMD) env GOPATH 2> /dev/null)
GOOS:=$(shell $(GOCMD) env GOOS 2> /dev/null)
GOARCH:=$(shell $(GOCMD) env GOARCH 2> /dev/null)
DOCKER_GO_VER=1.16

VERSION=$(shell git describe --tags --abbrev=0)
DATE=$(shell date +%FT%T%z)
VERSION_FORMATTED=$(shell git describe --tags --abbrev=0 | sed 's/\./-/g')

all: check-go install test build

check-go:
ifndef GOPATH
	$(error "go is not available, please install")
endif

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -r dist/
	rm -r builds/

deps:
	${GOCMD} get -v

install: clean deps

test: 
	$(GOTEST) -v ./...

reset-version:
	printf $(VERSION) > ./assets/.version


# Build
build: local-build
local-build:
	GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-${GOOS} -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-${GOOS}"

local-build-linux:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-linux -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-linux"

local-build-darwin:
	GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-darwin -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-darwin"

# Release
release-test:
	goreleaser release --skip-publish --skip-sign --rm-dist

check-current-tag-version:
	CURRENT_VERSION=$(shell git describe --abbrev=0 --tags)

check-increment-version:
	RELEASE_TAG=$(shell ${PWD}/increment_version.sh -p $(shell git describe --abbrev=0 --tags))

# Ref: https://github.com/fmahnke/shell-semver
release-patch:
	git fetch origin --tags --force
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -p $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./assets/.version
	git add ./assets/.version && git commit -m "Lagoon-sync $(VERSION) release"
	git tag $(VERSION) -f
	git push $(GIT_ORIGIN) main --tags

release-minor:
	git fetch origin --tags --force
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -m $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./assets/.version
	git add ./assets/.version && git commit -m "Lagoon-sync $(VERSION) release"
	git tag $(VERSION) -f
	git push $(GIT_ORIGIN) main --tags

release-major:
	git fetch origin --tags --force
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -M $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./assets/.version
	git add ./assets/.version && git commit -m "Lagoon-sync $(VERSION) release"
	git tag $(VERSION) -f	
	git push $(GIT_ORIGIN) main --tags
