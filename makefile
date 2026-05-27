.PHONY: run test add sub mul div chat

ARGS = $(filter-out $@,$(MAKECMDGOALS))

run:
	go run ./...
test:
	go test ./...
add sub mul div:
	go run ./client $@ $(ARGS)
chat:
	go run ./client chat

%:
	@:
