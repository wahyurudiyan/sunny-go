APP_NAME=sunny

.PHONY: build clean run

build:
	go build -o bin/$(APP_NAME) ./cmd

clean:
	rm -rf bin/
