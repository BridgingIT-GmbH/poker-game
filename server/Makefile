
GOLANG_CI_LINT_VERSION = 2.8.0
GOLANG_CI_LINT = bin/golangci-lint-$(GOLANG_CI_LINT_VERSION)

.PHONY: run/server
run/server:
	go run ./cmd/server/main.go

.PHONY: run/spectator
run/spectator:
	go run ./cmd/spectator/main.go

.PHONY: out
out:
	@mkdir -pv out

.PHONY: build
build: out
	go build -buildvcs=true -trimpath -o out ./...

.PHONY: test
test:
	go test ./...

.PHONY: bin
bin:
	@mkdir -pv bin

.PHONY: $(GOLANG_CI_LINT)
$(GOLANG_CI_LINT): bin
	@if [ ! -e "$(@)" ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b bin v$(GOLANG_CI_LINT_VERSION); \
		mv bin/golangci-lint "$(@)"; \
	fi

.PHONY: lint
lint: $(GOLANG_CI_LINT)
	$(GOLANG_CI_LINT) run --fix

.PHONY: lint/clean
lint/clean: $(GOLANG_CI_LINT)
	$(GOLANG_CI_LINT) cache clean

.PHONY: lint/ci
lint/ci: $(GOLANG_CI_LINT) lint/clean
	$(GOLANG_CI_LINT) run