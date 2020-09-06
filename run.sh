#!/bin/bash
set -e

# build ui
cd frontend
yarn build
cd ..

go run ./backend -t $DISCORD_TOKEN -p 1337 -d "$(pwd)"
