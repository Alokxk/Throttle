.PHONY: run build test clean migrate createdb docker-up docker-down docker-build

run:
	go run main.go

build:
	go build -o throttle .

test:
	go test ./... -v

clean:
	rm -f throttle

migrate:
	cp db/migrations/001_create_tables.sql /tmp/
	cp db/migrations/002_create_rules.sql /tmp/
	cp db/migrations/003_add_default_algorithm.sql /tmp/
	cp db/migrations/004_create_exemptions.sql /tmp/
	cp db/migrations/005_hash_api_keys.sql /tmp/
	sudo -u postgres psql -d throttle -f /tmp/001_create_tables.sql
	sudo -u postgres psql -d throttle -f /tmp/002_create_rules.sql
	sudo -u postgres psql -d throttle -f /tmp/003_add_default_algorithm.sql
	sudo -u postgres psql -d throttle -f /tmp/004_create_exemptions.sql
	sudo -u postgres psql -d throttle -f /tmp/005_hash_api_keys.sql

createdb:
	sudo -u postgres psql -c "CREATE DATABASE throttle;"

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

docker-build:
	docker-compose build