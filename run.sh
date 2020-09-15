#!/bin/bash
set -e

# build ui
cd frontend
bash build.sh
cd ..

go run ./backend -t $DISCORD_TOKEN -p 1337 -d "$(pwd)" \
	-spotify-id=$SPOTIFY_ID -spotify-secret=$SPOTIFY_SECRET \
	-video-dir="$(pwd)/videocache"
