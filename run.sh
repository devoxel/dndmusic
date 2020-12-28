#!/bin/bash
set -e

if [ -z "$NO_UI" ]
then
	# build ui
	cd frontend
	bash build.sh
	cd ..
fi

go run ./backend -t "$DISCORD_TOKEN" -p 9116 -d "$(pwd)" \
	-spotify-id="$SPOTIFY_ID" -spotify-secret="$SPOTIFY_TOKEN" \
	-video-dir="$(pwd)/videocache" --working-dir="$(pwd)"
