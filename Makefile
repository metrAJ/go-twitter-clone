.PHONY: help up down restart tidy

help:
	@echo "usage: make [target]"
	@echo ""
	@echo "targets:"
	@echo "  run		Run start.sh script. Recommended for the 1st use" 
	@echo "  up         Build and start the docker-compose cluster in the background"
	@echo "  down       Stop everything and wipe the local DB volumes"
	@echo "  restart    Nuke the containers and start fresh"
	@echo "  tidy       Format Go code and clean up go.mod"

run:
	@bash start.sh

up:
	@echo "Building and starting containers..."
	docker-compose up --build -d
	@echo "Done. The API should be coming up on port 3000 shortly."

down:
	@echo "Stopping containers and wiping data..."
	docker-compose down -v --remove-orphans

restart: down up

tidy:
	@echo "Formatting code and tidying modules..."
	go fmt ./...
	go mod tidy
