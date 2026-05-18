#!/bin/bash

set -e
echo "Booting up"

if [ ! -f .env ]; then
    echo "No .env file found!"
    echo "Automatically creating one from .env.example..."
    cp .env.example .env
fi

echo "Cleaning up previous instances"
docker-compose down -v --remove-orphans

echo "Building Go binaries and starting infrastructure..."
docker-compose up --build -d

echo "System is booting! May take 10-20s to initialize"

