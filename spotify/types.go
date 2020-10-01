package spotify

import (
	"errors"
	"reflect"
)

// Credit to github.com/zmb3/spotify which originally wrote these types and
// paging logic.

// basePage contains all of the fields in a Spotify paging object, except
// for the actual items.  This type is meant to be embedded in other types
// that add the Items field.
type basePage struct {
	// A link to the Web API Endpoint returning the full
	// result of this request.
	Endpoint string `json:"href"`
	// The maximum number of items in the response, as set
	// in the query (or default value if unset).
	Limit int `json:"limit"`
	// The offset of the items returned, as set in the query
	// (or default value if unset).
	Offset int `json:"offset"`
	// The total number of items available to return.
	Total int `json:"total"`
	// The URL to the next page of items (if available).
	Next string `json:"next"`
	// The URL to the previous page of items (if available).
	Previous string `json:"previous"`
}

// pageable is an internal interface for types that support paging
// by embedding basePage.
type pageable interface{ canPage() }

func (b basePage) canPage() {}

// ErrNoMorePages is the error returned when you attempt to get the next
// (or previous) set of data but you've reached the end of the data set.
var ErrNoMorePages = errors.New("spotify: no more pages")

// NextPage fetches the next page of items and writes them into p.
// It returns ErrNoMorePages if p already contains the last page.
func (c *Client) NextPage(p pageable) error {
	val := reflect.ValueOf(p).Elem()
	field := val.FieldByName("Next")
	nextURL := field.Interface().(string)

	if len(nextURL) == 0 {
		return ErrNoMorePages
	}

	// Zero out the page so that we can overwrite it in the next
	// call to get. This is necessary because encoding/json does
	// not clear out existing values when unmarshaling JSON null.
	zero := reflect.Zero(val.Type())
	val.Set(zero)

	return c.get(nextURL, p)
}

// PreviousPage fetches the previous page of items and writes them into p.
// It returns ErrNoMorePages if p already contains the last page.
func (c *Client) PreviousPage(p pageable) error {
	val := reflect.ValueOf(p).Elem()
	field := val.FieldByName("Previous")
	prevURL := field.Interface().(string)

	if len(prevURL) == 0 {
		return ErrNoMorePages
	}

	// Zero out the page so that we can overwrite it in the next
	// call to get. This is necessary because encoding/json does
	// not clear out existing values when unmarshaling JSON null.
	zero := reflect.Zero(val.Type())
	val.Set(zero)

	return c.get(prevURL, p)
}

// PlaylistTrack contains info about a track in a playlist.
type PlaylistTrack struct {
	// The date and time the track was added to the playlist.
	// You can use the TimestampLayout constant to convert
	// this field to a time.Time value.
	// Warning: very old playlists may not populate this value.
	AddedAt string `json:"added_at"`
	// The Spotify user who added the track to the playlist.
	// Warning: vary old playlists may not populate this value.
	AddedBy User `json:"added_by"`
	// Whether this track is a local file or not.
	IsLocal bool `json:"is_local"`
	// Information about the track.
	Track FullTrack `json:"track"`
}

// SimpleTrack contains basic info about a track.
type SimpleTrack struct {
	Artists []SimpleArtist `json:"artists"`
	// A list of the countries in which the track can be played,
	// identified by their ISO 3166-1 alpha-2 codes.
	AvailableMarkets []string `json:"available_markets"`
	// The disc number (usually 1 unless the album consists of more than one disc).
	DiscNumber int `json:"disc_number"`
	// The length of the track, in milliseconds.
	Duration int `json:"duration_ms"`
	// Whether or not the track has explicit lyrics.
	// true => yes, it does; false => no, it does not.
	Explicit bool `json:"explicit"`
	// External URLs for this track.
	ExternalURLs map[string]string `json:"external_urls"`
	// A link to the Web API endpoint providing full details for this track.
	Endpoint string `json:"href"`
	ID       ID     `json:"id"`
	Name     string `json:"name"`
	// A URL to a 30 second preview (MP3) of the track.
	PreviewURL string `json:"preview_url"`
	// The number of the track.  If an album has several
	// discs, the track number is the number on the specified
	// DiscNumber.
	TrackNumber int `json:"track_number"`
	URI         URI `json:"uri"`
}

// SimpleAlbum contains basic data about an album.
type SimpleAlbum struct {
	// The name of the album.
	Name string `json:"name"`
	// A slice of SimpleArtists
	Artists []SimpleArtist `json:"artists"`
	// The field is present when getting an artist’s
	// albums. Possible values are “album”, “single”,
	// “compilation”, “appears_on”. Compare to album_type
	// this field represents relationship between the artist
	// and the album.
	AlbumGroup string `json:"album_group"`
	// The type of the album: one of "album",
	// "single", or "compilation".
	AlbumType string `json:"album_type"`
	// The SpotifyID for the album.
	ID ID `json:"id"`
	// The SpotifyURI for the album.
	URI URI `json:"uri"`
	// The markets in which the album is available,
	// identified using ISO 3166-1 alpha-2 country
	// codes.  Note that al album is considered
	// available in a market when at least 1 of its
	// tracks is available in that market.
	AvailableMarkets []string `json:"available_markets"`
	// A link to the Web API enpoint providing full
	// details of the album.
	Endpoint string `json:"href"`
	// The cover art for the album in various sizes,
	// widest first.
	Images []Image `json:"images"`
	// Known external URLs for this album.
	ExternalURLs map[string]string `json:"external_urls"`
	// The date the album was first released.  For example, "1981-12-15".
	// Depending on the ReleaseDatePrecision, it might be shown as
	// "1981" or "1981-12". You can use ReleaseDateTime to convert this
	// to a time.Time value.
	ReleaseDate string `json:"release_date"`
	// The precision with which ReleaseDate value is known: "year", "month", or "day"
	ReleaseDatePrecision string `json:"release_date_precision"`
}

// SimpleArtist contains basic info about an artist.
type SimpleArtist struct {
	Name string `json:"name"`
	ID   ID     `json:"id"`
	// The Spotify URI for the artist.
	URI URI `json:"uri"`
	// A link to the Web API enpoint providing full details of the artist.
	Endpoint     string            `json:"href"`
	ExternalURLs map[string]string `json:"external_urls"`
}

// FullTrack provides extra track data in addition to what is provided by SimpleTrack.
type FullTrack struct {
	SimpleTrack
	// The album on which the track appears. The album object includes a link in href to full information about the album.
	Album SimpleAlbum `json:"album"`
	// Known external IDs for the track.
	ExternalIDs map[string]string `json:"external_ids"`
	// Popularity of the track.  The value will be between 0 and 100,
	// with 100 being the most popular.  The popularity is calculated from
	// both total plays and most recent plays.
	Popularity int `json:"popularity"`
}

// PlaylistTrackPage contains information about tracks in a playlist.
type PlaylistTrackPage struct {
	basePage
	Tracks []PlaylistTrack `json:"items"`
}

// ID is a base-62 identifier for an artist, track, album, etc.
// It can be found at the end of a spotify.URI.
type ID string

// URI identifies an artist, album, track, or category.  For example,
// spotify:track:6rqhFgbbKwnb9MLmUQDhG6
type URI string

// User contains the basic, publicly available information about a Spotify user.
type User struct {
	// The name displayed on the user's profile.
	// Note: Spotify currently fails to populate
	// this field when querying for a playlist.
	DisplayName string `json:"display_name"`
	// Known public external URLs for the user.
	ExternalURLs map[string]string `json:"external_urls"`
	// Information about followers of the user.
	Followers Followers `json:"followers"`
	// A link to the Web API endpoint for this user.
	Endpoint string `json:"href"`
	// The Spotify user ID for the user.
	ID string `json:"id"`
	// The user's profile image.
	Images []Image `json:"images"`
	// The Spotify URI for the user.
	URI URI `json:"uri"`
}

// Followers contains information about the number of people following a
// particular artist or playlist.
type Followers struct {
	// The total number of followers.
	Count uint `json:"total"`
	// A link to the Web API endpoint providing full details of the followers,
	// or the empty string if this data is not available.
	Endpoint string `json:"href"`
}

// Image identifies an image associated with an item.
type Image struct {
	// The image height, in pixels.
	Height int `json:"height"`
	// The image width, in pixels.
	Width int `json:"width"`
	// The source URL of the image.
	URL string `json:"url"`
}

// PlaylistTracks contains details about the tracks in a playlist.
type PlaylistTracks struct {
	// A link to the Web API endpoint where full details of
	// the playlist's tracks can be retrieved.
	Endpoint string `json:"href"`
	// The total number of tracks in the playlist.
	Total uint `json:"total"`
}

// SimplePlaylist contains basic info about a Spotify playlist.
type SimplePlaylist struct {
	// Indicates whether the playlist owner allows others to modify the playlist.
	// Note: only non-collaborative playlists are currently returned by Spotify's Web API.
	Collaborative bool              `json:"collaborative"`
	ExternalURLs  map[string]string `json:"external_urls"`
	// A link to the Web API endpoint providing full details of the playlist.
	Endpoint string `json:"href"`
	ID       ID     `json:"id"`
	// The playlist image.  Note: this field is only  returned for modified,
	// verified playlists. Otherwise the slice is empty.  If returned, the source
	// URL for the image is temporary and will expire in less than a day.
	Images   []Image `json:"images"`
	Name     string  `json:"name"`
	Owner    User    `json:"owner"`
	IsPublic bool    `json:"public"`
	// The version identifier for the current playlist. Can be supplied in other
	// requests to target a specific playlist version.
	SnapshotID string `json:"snapshot_id"`
	// A collection to the Web API endpoint where full details of the playlist's
	// tracks can be retrieved, along with the total number of tracks in the playlist.
	Tracks PlaylistTracks `json:"tracks"`
	URI    URI            `json:"uri"`
}

// SimplePlaylistPage contains SimplePlaylists returned by the Web API.
type SimplePlaylistPage struct {
	basePage
	Playlists []SimplePlaylist `json:"items"`
}

// FullPlaylist provides extra playlist data in addition to the data provided by SimplePlaylist.
type FullPlaylist struct {
	SimplePlaylist
	// The playlist description.  Only returned for modified, verified playlists.
	Description string `json:"description"`
	// Information about the followers of this playlist.
	Followers Followers         `json:"followers"`
	Tracks    PlaylistTrackPage `json:"tracks"`
}
