CONFIG_PATH=${HOME}/.distsql

.PHONY: compile
compile:
	protoc proto/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

.PHONY: gencert
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \

	test/server-csr.json | cfssljson -bare server

.PHONY: test
test:
	go test -race ./...
