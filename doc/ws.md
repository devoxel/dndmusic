# websocket api 

On connection server should wait for StatusCheck message. Then
the server should report a status and write the password required
if that status is "Unverified".

If the status is Verified then the server should return a Music UI
state - a list of playlists to render on the UI to be selected by the
user. If a user clicks on a playlist, the client should send a MusicSelect
message, which prompts the bot to start playing music.

## StatusCheck (sent by client) Ask the server for a status

### Request

{
       "message": "StatusCheck",
}

### Responses

{
       "message": "StatusCheckResponse",
       "status": "Verified"
       "playlists": []string{}...
       "nowPlaying": {
               "playlist": "https://.../",
               "song": "Living La Vida Loca",
       },
},
{
       "message": "StatusCheckResponse",
       "status": "Unverified"
       "password": "123123",
}


## Music Selection

### Request

{
       "message": "MusicSelect",
       "type": "Playlist",
       "playlist": "https://.../",
}

{
       "message": "MusicSelect",
       "type": "SkipSong",
}

{
       "message": "MusicSelect",
       "type": "SetSong",
       "song": "",
}
