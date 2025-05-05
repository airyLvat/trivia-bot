#!/bin/bash

IMAGE_NAME="trivia-bot"
CONTAINER_NAME="trivia-bot"
VOLUME_NAME="trivia-bot-data"
SCRIPT_DIR="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
SOURCE_DB="$SCRIPT_DIR/trivia.db"
CONTAINER_DB_PATH="/app/data/trivia.db"

if [ ! -f "$SOURCE_DB" ]; then
    echo "Error: $SOURCE_DB not found. Please place trivia.db in ~/bots/trivia-bot/."
    exit 1
fi

if ! docker volume inspect "$VOLUME_NAME" >/dev/null 2>&1; then
    echo "Creating volume $VOLUME_NAME..."
    docker volume create "$VOLUME_NAME"
fi

VOLUME_PATH=$(docker volume inspect "$VOLUME_NAME" --format '{{ .Mountpoint }}')
DB_PATH="$VOLUME_PATH/trivia.db"

INITIALIZE_DB=false
if [ -f "$DB_PATH" ]; then
    QUESTION_COUNT=$(sudo sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM questions" 2>/dev/null)
    if [ $? -eq 0 ] && [ "$QUESTION_COUNT" -gt 0 ]; then
        echo "Volume database ($DB_PATH) has $QUESTION_COUNT questions. Skipping initialization."
    else
        echo "Volume database is empty or invalid. Will initialize."
        INITIALIZE_DB=true
    fi
else
    echo "No database found in volume. Will initialize."
    INITIALIZE_DB=true
fi

echo "Building Docker image $IMAGE_NAME..."
docker build -t "$IMAGE_NAME" . || { echo "Build failed."; exit 1; }

echo "Stopping and removing existing container (if any)..."
docker stop "$CONTAINER_NAME" >/dev/null 2>&1
docker rm "$CONTAINER_NAME" >/dev/null 2>&1

echo "Starting container $CONTAINER_NAME..."
docker run -d --restart unless-stopped --name "$CONTAINER_NAME" -v "$VOLUME_NAME:/app/data" --env DATABASE_PATH="$CONTAINER_DB_PATH" "$IMAGE_NAME" || { echo "Failed to start container."; exit 1; }

if [ "$INITIALIZE_DB" = true ]; then
    echo "Copying $SOURCE_DB to container..."
    docker cp "$SOURCE_DB" "$CONTAINER_NAME:$CONTAINER_DB_PATH" || { echo "Failed to copy database."; exit 1; }
    echo "Restarting container to apply database..."
    docker restart "$CONTAINER_NAME" >/dev/null
fi

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
