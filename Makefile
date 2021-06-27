gen-proto:
	protoc --go_out=./ pkg/playstore/playstore.proto
install-cli:
	go get -v github.com/jarijaas/go-gplayapi/cmd/gplaycli