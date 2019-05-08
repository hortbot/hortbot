#!/bin/sh -e

cd "${0%/*}"

echo "Starting database"
DOCKER_ID=$(docker run --rm -p 5432:5432 -d zikaeroh/postgres-initialized)

function kill_container {
    echo "Killing database"
    docker kill $DOCKER_ID > /dev/null
}

trap kill_container EXIT

echo "Waiting for container to be ready"
while ! curl http://localhost:5432 2>&1 | grep -q '52'; do sleep 1; done

echo "Migrating database up"
migrate -database 'postgres://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable' -path internal/db/migrations up

echo "Generating models"
sqlboiler psql --wipe --no-hooks --no-rows-affected --no-tests

echo "Generating migration data"
go generate ./internal/db/migrations
