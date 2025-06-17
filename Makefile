# Makefile

.PHONY: build run docker-build docker-run clean

build:
	go build -o kubejobs ./cmd/server

run: build
	./kubejobs

docker-build:
	docker build -t kubejobs .

docker-run:
	docker run -p 8080:8080 kubejobs