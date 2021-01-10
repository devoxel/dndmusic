# flarhgunnstow

flarhgunnstow is a discord music bot with full playlist integration and a websocket web UI.

## Usage

This bot is only available for you to use self hosted.

## Warning

This bot is pre-alpha, and contains bugs & issues, and a lot of what is in this README is
unimplemented. See [TODO](#TODO) for a better list.

These docs are not written for beginners. Eventually I'll get around to writing an easy
deployment guide. Given how buggy it is currently you may just have to wait for it.

## Features

The WebUI allows DMs to interact with the bot without having to type, and gives quick
access to music when you need it most.

As a consequence of this, it also functions as a great music bot that remembers your
playlists!

## Layout

This project is in two main sections:

- `backend` - contains the discord bot 
- `frontend` - contains the frontend UI code

The frontend code is built first (see `run.sh`) and that is hosted by the bot.

## Hosting

Set `$DISCORD_TOKEN` and use `run.sh` to start the bot. You need to have
`ffmpeg`, `youtube-dl`, the go toolchain, and the nodejs toolchain.

I'll eventually make a binary release but for now no dice.

Here's a gotcha: when running this bot through nginx you have to ensure you
properly redirect '/ws' headers (see below example):

```
server {
        server_name $UR_SERVER_HERE;

        location /ws {
                proxy_pass http://127.0.0.1:9116;
                proxy_http_version 1.1;
                proxy_set_header Upgrade $http_upgrade;
                proxy_set_header Connection "Upgrade";
                proxy_set_header Host $host;
        }

        location / {
                proxy_pass http://127.0.0.1:9116;
        }
}
```

## TODO

Urgent stuff to move the bot into alpha:

- Backend: Persist guild playlists
- Audio: Allow building queue from youtube playlist
- Web UI: housekeeping, lots of old artifacts from early tests
- Web UI: Add playlist creation

Eventual stuff:

- Docs: Write a tutorial on how to set it up.
- Discord: implement commands from [API](#wanted-api)
- Web UI: Combat mode (quickly switch to combat music)
- Web UI: Allow full bot interactions & help screen
- Web UI: Stop polling, and start getting updates in REAL TIME

I could convert these to issues probably but I'll only do this if people are interested in contributing.

### Wanted API

```
playlist add name [url] | add a playlist
  or pl
playlist [list]         | list playlists
play [playlist]         | play playlist
play [url]              | queue song to ephemeral playlist ("Now Playing")
play [playlist_url]     | queue all songs in playlist to ephemeral playlist
playnext [url]          | add song to ephermal playlist after currently playing song
 or play_next
 or pn
save_current [name]     | saves all songs in ephemeral playlist to a new playlist
remove N                | remove Nth song in ephemeral playlist
remove N..M             | remove a group of songs from playlist
remove_all [user]       | clears all of a certain person's additions to the playlist (maybe shouldn't add this one)
ui                      | create a UI session where you can control this stuff on your browser
  or party
  or ...
queue balanced          | set queue mode to balanced, round robin style playing.
queue normal            | set queue mode to expected
pause                   | if playing, pause
play                    | if paused, play
clear                   | clear now playing
bounce                  | bot exits the channel
  or exit
queue                   | list all songs
  or q
```
