build:
	go build -o bin/shortner

run: build
	./bin/shortner