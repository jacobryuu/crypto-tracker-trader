.PHONY: build run docker-up

build:
\tgo build -o bin/server ./cmd/server

run:
\tgo run ./cmd/server

docker-up:
\tdocker-compose up --build
