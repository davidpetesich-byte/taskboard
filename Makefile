.PHONY: build dev frontend clean install

BUILD_DIR := cmd/taskboard
BINARY := taskboard

build: frontend
	go build -o $(BINARY) ./$(BUILD_DIR)

dev:
	go run ./$(BUILD_DIR) start --foreground

frontend:
	cd web && npm install && npm run build
	mkdir -p $(BUILD_DIR)/web/dist
	cp -r web/dist/* $(BUILD_DIR)/web/dist/

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)/web
	rm -rf web/dist web/node_modules

# Copy then rename so a running daemon or MCP process keeps its old inode
# instead of having a mapped, code-signed binary overwritten in place.
install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY).tmp
	mv -f $(HOME)/.local/bin/$(BINARY).tmp $(HOME)/.local/bin/$(BINARY)

dev-frontend:
	cd web && npm run dev

test:
	go test ./internal/...
