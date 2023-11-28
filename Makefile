.PHONY: install
install:
	@bash build.sh
	@cp ./bin/grfs ${HOME}/go/bin

.PHONY: test
test:
	@go test ./...
