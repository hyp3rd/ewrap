include .project-settings.env

GOLANGCI_LINT_VERSION ?= v2.7.1
GO_VERSION ?= 1.25.5
GCI_PREFIX ?= github.com/hyp3rd/ewrap
PROTO_ENABLED ?= true

GOFILES = $(shell find . -type f -name '*.go' -not -path "./pkg/api/*" -not -path "./vendor/*" -not -path "./.gocache/*" -not -path "./.git/*")

test:
	go test -v -timeout 5m -cover ./...

test-race:
	go test -race ./...

benchmark:
	go test -bench=. -benchmem ./pkg/ewrap
	go test -bench=Benchmark -benchmem ./test
	# go test -run=TestProfile -cpuprofile=cpu.prof -memprofile=mem.prof ./test

update-deps:
	go get -v -u ./...
	go mod tidy

prepare-toolchain:
	$(call check_command_exists,docker) || (echo "Docker is missing, install it before starting to code." && exit 1)

	$(call check_command_exists,git) || (echo "git is not present on the system, install it before starting to code." && exit 1)

	$(call check_command_exists,go) || (echo "golang is not present on the system, download and install it at https://go.dev/dl" && exit 1)

	@echo "Installing gci...\n"
	$(call check_command_exists,gci) || go install github.com/daixiang0/gci@latest

	@echo "Installing gofumpt...\n"
	$(call check_command_exists,gofumpt) || go install mvdan.cc/gofumpt@latest

	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)...\n"
	$(call check_command_exists,golangci-lint) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" $(GOLANGCI_LINT_VERSION)

	@echo "Installing staticcheck...\n"
	$(call check_command_exists,staticcheck) || go install honnef.co/go/tools/cmd/staticcheck@latest

	@echo "Installing govulncheck...\n"
	$(call check_command_exists,govulncheck) || go install golang.org/x/vuln/cmd/govulncheck@latest

	@echo "Installing gosec...\n"
	$(call check_command_exists,gosec) || go install github.com/securego/gosec/v2/cmd/gosec@latest

	@echo "Checking if pre-commit is installed..."
	pre-commit --version || (echo "pre-commit is not installed, install it with 'pip install pre-commit'" && exit 1)

	@echo "Initializing pre-commit..."
	pre-commit validate-config || pre-commit install && pre-commit install-hooks

update-toolchain:
	@echo "Updating gci...\n"
	go install github.com/daixiang0/gci@latest

	@echo "Updating gofumpt...\n"
	go install mvdan.cc/gofumpt@latest

	@echo "Updating govulncheck...\n"
	go install golang.org/x/vuln/cmd/govulncheck@latest

	@echo "Updating gosec...\n"
	go install github.com/securego/gosec/v2/cmd/gosec@latest

	@echo "Updating staticcheck...\n"
	go install honnef.co/go/tools/cmd/staticcheck@latest

lint: prepare-toolchain
	@for file in ${GOFILES}; do \
		gci write -s standard -s default -s blank -s dot -s "prefix($(GCI_PREFIX))" -s localmodule --skip-vendor --skip-generated $$file; \
	done

	@echo "\nRunning gofumpt..."
	gofumpt -l -w ${GOFILES}

	@echo "\nRunning staticcheck..."
	staticcheck ./...

	@echo "\nRunning golangci-lint $(GOLANGCI_LINT_VERSION)..."
	golangci-lint run -v --fix ./...

vet:
	@echo "Running go vet..."

	$(call check_command_exists,shadow) || go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest

	@for file in ${GOFILES}; do \
		go vet -vettool=$(shell which shadow) $$file; \
	done

sec:
	@echo "Running govulncheck..."
	govulncheck ./...

	@echo "\nRunning gosec..."
	gosec -exclude-generated ./...

# check_command_exists is a helper function that checks if a command exists.
define check_command_exists
@which $(1) > /dev/null 2>&1 || (echo "$(1) command not found" && exit 1)
endef

ifeq ($(call check_command_exists,$(1)),false)
  $(error "$(1) command not found")
endif

# help prints a list of available targets and their descriptions.
help:
	@echo "Available targets:"
	@echo
	@echo "test\t\t\t\tRun all tests in the project."
	@echo "update-deps\t\t\tUpdate all dependencies in the project."
	@echo "prepare-toolchain\t\tPrepare the development toolchain by installing necessary tools."
	@echo "update-toolchain\t\tUpdate the development toolchain tools to their latest versions."
	@echo "benchmark\t\t\tRun benchmarks for the project."
	@echo "sec\t\t\t\tRun the govulncheck and gosec security analysis tools on all packages in the project."
	@echo "vet\t\t\t\tRun go vet and shadow analysis on all packages in the project."
	@echo "lint\t\t\t\tRun the staticcheck and golangci-lint static analysis tools on all packages in the project."
	@echo
	@echo "help\t\t\t\tPrint this help message."
	@echo
	@echo "For more information, see the project README."

.PHONY: prepare-toolchain update-toolchain sec vet test benchmark update-deps lint help
