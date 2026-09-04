GO_FILES := $(shell find cmd internal web -type f -name '*.go' -print)

.PHONY: \
	fmt fmt-check \
	mod-check \
	lint typecheck test vuln \
	build \
	ci

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "The following Go files are not formatted:" >&2; \
		echo "$$files" >&2; \
		exit 1; \
	fi

mod-check:
	go mod tidy -diff
	go mod verify
	go list -mod=readonly ./... >/dev/null

lint:
	go vet ./...

typecheck:
	go build ./...

test:
	go test ./...

vuln:
	govulncheck ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o bin/eif ./cmd/eif

ci: fmt-check mod-check lint typecheck test build
