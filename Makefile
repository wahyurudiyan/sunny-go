APP_NAME=sunny

.PHONY: build clean run

build:
	go build -o bin/$(APP_NAME) ./cmd

run: build
	./bin/$(APP_NAME)

clean:
	rm -rf bin/
