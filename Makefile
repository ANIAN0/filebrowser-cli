# filebrowser-cli Makefile
# 支持平台：Linux / macOS / Windows (Git Bash / WSL)

CLI_NAME := filebrowser-cli
BIN_DIR := bin
DIST_DIR := dist
GO ?= go
GOBIN_PATH := $(shell $(GO) env GOPATH)/bin

.PHONY: build install uninstall test build-all clean help

## help: 显示帮助信息
help:
	@echo "filebrowser-cli Makefile"
	@echo ""
	@echo "用法：make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build       构建 CLI 到 $(BIN_DIR)/"
	@echo "  install     安装到 $(GOBIN_PATH)/（需先 build）"
	@echo "  uninstall   从 $(GOBIN_PATH)/ 卸载"
	@echo "  test        运行所有单元测试"
	@echo "  build-all   交叉编译 6 个平台到 $(DIST_DIR)/"
	@echo "  clean       清理构建产物"
	@echo "  help        显示本帮助信息"

## build: 构建 CLI
build:
	@echo "Building $(CLI_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(CLI_NAME) .
	@echo "Build complete: $(BIN_DIR)/$(CLI_NAME)"

## install: 安装到 GOPATH/bin
install: build
	@echo "Installing $(CLI_NAME) to $(GOBIN_PATH)..."
	@mkdir -p $(GOBIN_PATH)
	install -m 0755 $(BIN_DIR)/$(CLI_NAME) $(GOBIN_PATH)/$(CLI_NAME)
	@echo "Installed to $(GOBIN_PATH)"
	@echo ""
	@echo "$$PATH" | tr ':' '\n' | grep -q "^$$(echo $(GOBIN_PATH) | sed 's|/|\\/|g')$$" || \
		echo "WARN: $(GOBIN_PATH) is not in your PATH." && \
		echo "      Add the following to your shell profile:" && \
		echo "        export PATH=\"$(GOBIN_PATH):\$$PATH\""

## uninstall: 从 GOPATH/bin 卸载
uninstall:
	@echo "Uninstalling $(CLI_NAME) from $(GOBIN_PATH)..."
	@if [ -f "$(GOBIN_PATH)/$(CLI_NAME)" ]; then \
		echo "  Removing $(CLI_NAME)..."; \
		rm -f $(GOBIN_PATH)/$(CLI_NAME); \
	else \
		echo "  $(CLI_NAME) not found, skipping..."; \
	fi
	@echo "Uninstall complete"

## test: 运行所有单元测试
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## build-all: 交叉编译 6 个平台
build-all:
	@echo "Cross-compiling for 6 platforms..."
	@mkdir -p $(DIST_DIR)
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=$$( [ "$$os" = "windows" ] && echo .exe || echo "" ); \
			echo "  Building $(CLI_NAME)-$$os-$$arch$$ext..."; \
			GOOS=$$os GOARCH=$$arch $(GO) build \
				-o $(DIST_DIR)/$(CLI_NAME)-$$os-$$arch$$ext . || exit 1; \
		done; \
	done
	@echo "Cross-compilation complete: $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

## clean: 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR) $(DIST_DIR)
	@echo "Clean complete"
