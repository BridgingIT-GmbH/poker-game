
.PHONY: run
run:
	go run ./main.go $(ARGS)

.PHONY: out
out:
	@mkdir -pv out

.PHONY: build
build: out
	go build -buildvcs=true -trimpath -o out ./...

.PHONY: test
test:
	go test ./...

