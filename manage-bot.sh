#!/bin/bash

SCRIPT_DIR="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"

IMAGE_NAME="trivia-bot"
CONTAINER_NAME="trivia-bot"
HOST_DATA_DIR="$SCRIPT_DIR/data"
CONTAINER_DB_PATH="/app/data/trivia.db"

if [ ! -d "$HOST_DATA_DIR" ]; then
    echo "Error: $HOST_DATA_DIR not found. Please create it and place trivia.db inside."
    exit 1
fi

if [ ! -f "$HOST_DATA_DIR/trivia.db" ]; then
    echo "Error: $HOST_DATA_DIR/trivia.db not found. Please place trivia.db in $HOST_DATA_DIR/"
    exit 1
fi

echo "Building Docker image $IMAGE_NAME..."
docker build -t "$IMAGE_NAME" . || { echo "Build failed."; exit 1; }

echo "Stopping and removing existing container (if any)..."
docker stop "$CONTAINER_NAME" >/dev/null 2>&1
docker rm "$CONTAINER_NAME" >/dev/null 2>&1

echo "Starting container $CONTAINER_NAME..."
docker run -d --name "$CONTAINER_NAME" \
    -v "$HOST_DATA_DIR:/app/data" \
    --env DATABASE_PATH="$CONTAINER_DB_PATH" \
    "$IMAGE_NAME" || { echo "Failed to start container."; exit 1; }

echo "Verifying database..."
QUESTION_COUNT=$(docker exec "$CONTAINER_NAME" sqlite3 "$CONTAINER_DB_PATH" "SELECT COUNT(*) FROM questions" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "Database has $QUESTION_COUNT questions."
else
    echo "Error: Failed to query database."
    exit 1
fi

echo "Checking container logs..."
docker logs "$CONTAINER_NAME" --tail 10

echo "Bot is running! Check Discord with !!trivia list."
