#!/bin/bash

# Load environment variables from .env file
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo ".env file not found."
  exit 1
fi

# Check if required variables are set
if [ -z "$WEBHOOK_URL" ] || [ -z "$TARGET_URL" ]; then
  echo "WEBHOOK_URL and TARGET_URL must be set in the .env file."
  exit 1
fi

echo "Starting Go server..."
go run main.go &
GO_PID=$!

sleep 2

echo "Starting Smee client..."
npx smee-client -u "$WEBHOOK_URL" -t "$TARGET_URL" &
SMEE_PID=$!

trap "echo 'Stopping...'; kill $GO_PID $SMEE_PID" INT
wait
