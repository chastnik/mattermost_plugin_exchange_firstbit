## Server
ifneq ($(HAS_SERVER),)
	include build/server.mk
endif

## Webapp
ifneq ($(HAS_WEBAPP),)
	include build/webapp.mk
endif

## Shim
install-go-deps:

check-style: gofmt govet ## Runs govet and gofmt against all packages.

## Apply
.PHONY: apply
apply:
ifneq ($(HAS_SERVER),)
	go get github.com/mattermost/mattermost-plugin-api
endif

.PHONY: fmt
fmt: gofmt ## Formats the code.

.PHONY: gofmt
gofmt: ## Runs gofmt against all packages.
ifneq ($(HAS_SERVER),)
	@echo Running gofmt
	@for package in $$(go list ./server/...); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "gofmt failure"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo gofmt success
endif

.PHONY: govet
govet: ## Runs govet against all packages.
ifneq ($(HAS_SERVER),)
	@echo Running govet
	$(GO) vet ./server/...
	@echo govet success
endif 