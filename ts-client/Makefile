.PHONY: install
install:
	npm install

.PHONY: build
build: install
	npx tsc

.PHONY: run
run: build
	node dist/main.js $(ARGS)

.PHONY: clean
clean:
	rm -rf dist node_modules
