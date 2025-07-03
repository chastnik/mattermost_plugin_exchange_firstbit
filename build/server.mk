## Builds the server, if it exists, for all supported architectures.
.PHONY: server
server: server/dist

server/dist: $(GO_FILES)
	cd server && echo "module github.com/mattermost/exchange-plugin/server\ngo 1.19" > go.mod
	cd server && $(GO) get github.com/mattermost/mattermost-server/v6@v6.7.2
	cd server && $(GO) get github.com/gorilla/mux@v1.8.1
	cd server && $(GO) mod tidy
	mkdir -p server/dist;
ifeq ($(MM_DEBUG),)
	cd server && env CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GO_BUILD_FLAGS) -trimpath -ldflags "-s -w" -o dist/plugin-linux-amd64;
	cd server && env CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -ldflags "-s -w" -o dist/plugin-darwin-amd64;
	cd server && env CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -ldflags "-s -w" -o dist/plugin-windows-amd64.exe;
else
	$(info DEBUG mode is on; to disable, unset MM_DEBUG)
	cd server && env CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -o dist/plugin-linux-amd64;
	cd server && env CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -o dist/plugin-darwin-amd64;
	cd server && env CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -o dist/plugin-windows-amd64.exe;
endif 