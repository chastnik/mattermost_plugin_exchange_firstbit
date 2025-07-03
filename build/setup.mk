PLUGIN_ID ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json id)
PLUGIN_VERSION ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json version)
HAS_SERVER ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json has_server)
HAS_WEBAPP ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json has_webapp)
MIN_SERVER_VERSION ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json min_server_version)
SERVER_EXECUTABLE ?= $(shell cat $(MANIFEST_FILE) | $(GO) run ./build/bin/json executable)

# Fallbacks for if we can't read from the manifest
PLUGIN_ID ?= com.mattermost.exchange-plugin
PLUGIN_VERSION ?= 1.0.0
HAS_SERVER ?= 1
HAS_WEBAPP ?= 1
MIN_SERVER_VERSION ?= 6.0.0
SERVER_EXECUTABLE ?= plugin-linux-amd64 