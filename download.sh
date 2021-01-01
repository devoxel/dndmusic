youtube-dl \
	--add-header "Accept-Language: \"en-US,en;q=0.5\"" \
	--cookies "cookies.txt" \
	--socket-timeout "10" \
	--default-search "auto" \
	--no-playlist \
	--no-call-home \
	--no-progress \
	--format "bestaudio" \
	$1 \
	-o - \
	| ffmpeg -i pipe:0 -f s16le -ar 48000 -ac 2 pipe:1
