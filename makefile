.PHONY: run test add sub mul div chat health radd rsub rmul rdiv rhealth rchat

ARGS = $(filter-out $@,$(MAKECMDGOALS))

run:
	go run ./...
test:
	go test ./...
add sub mul div:
	go run ./client $@ $(ARGS)
chat:
	go run ./client chat
health:
	go run ./client health
radd rsub rmul rdiv:
	ENABLE_ETCD_RESOLVER=true go run ./client $(patsubst r%,%,$@) $(ARGS)
rhealth:
	ENABLE_ETCD_RESOLVER=true go run ./client health

rchat:
	ENABLE_ETCD_RESOLVER=true go run ./client chat
%:
	@:
