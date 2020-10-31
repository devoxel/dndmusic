#!/bin/bash
set -e

if [ -z "$NO_UI" ]
then
	# build ui
	cd frontend
	bash build.sh
	cd ..
fi

go run ./backend -t $DISCORD_TOKEN -p 1337 -d "$(pwd)" \
	-spotify-id=$SPOTIFY_ID -spotify-secret=$SPOTIFY_SECRET \
	-video-dir="$(pwd)/videocache" --working-dir="$(pwd)"
