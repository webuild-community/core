DB_CONTAINER?=webuild_core_db

remove-infras:
	docker-compose stop

init: remove-infras
	docker-compose up -d
	@echo "Waiting for database connection..."
	@while ! docker exec $(DB_CONTAINER) pg_isready -h localhost -p 5432 > /dev/null; do \
		sleep 1; \
	done

.PHONY: run
run:
	go run cmd/*.go

.PHONY: test
test:
	go test ./...

		