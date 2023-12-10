protogen:
	./scripts/protogen.sh

build:
	go build -o bin/main main.go

debug: build
	./bin/main

filter:
	cat output.log | grep "info" > info.log
	cat output.log | grep "debug" > debug.log