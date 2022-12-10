
clean:
	rm -rf bin/

build:
	go build  -o bin/aggregation-service github.com/D4niel44/upfluence-challenge/cmd/aggregation-service

test:
	make clean
	go test -v -timeout 30s -count=1 github.com/D4niel44/upfluence-challenge/...
