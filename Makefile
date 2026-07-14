.PHONY: run build test clean migrate-up migrate-down createdb docker-up docker-down docker-build

-include .env
export

run:
	go run main.go

build:
	go build -o throttle .

test:
	go test ./... -v

clean:
	rm -f throttle

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

createdb:
	sudo -u postgres psql -c "CREATE DATABASE throttle;"

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

docker-build:
	docker-compose build