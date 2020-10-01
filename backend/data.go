package main

/*
	page, err := adm.s.GetUserPlaylists("bezoing")
	if err != nil {
		panic(err)
	}

	f := `
	{
		Title:    "%s",
		URL:      "https://open.spotify.com/playlist/%s",
		Category: "%s",
	},`

	for {
		for _, pl := range page.Playlists {
			b := strings.SplitN(pl.Name, ":", 2)
			fmt.Printf(f, pl.Name, pl.ID, b[0])
		}

		err = adm.s.NextPage(page)
		if err == spotify.ErrNoMorePages {
			os.Exit(0)
		} else if err != nil {
			panic(err)
		}

	}
*/

var samplePlaylists = []WSPlaylist{
	{
		Title:    "Ambient: Cavern",
		URL:      "https://open.spotify.com/playlist/7cgECSzxFYwjHugNdbur1O",
		Category: "Ambient",
	},
	{
		Title:    "Ambient: Forest",
		URL:      "https://open.spotify.com/playlist/5ayvxbK8CveLIj4llcibs2",
		Category: "Ambient",
	},
	{
		Title:    "Ambient: Mountain Pass",
		URL:      "https://open.spotify.com/playlist/4y88W8yD8M32PJ4ZNJVzAp",
		Category: "Ambient",
	},
	{
		Title:    "Ambient: Mystical",
		URL:      "https://open.spotify.com/playlist/47JbzbE2fpng1VU0VeufGU",
		Category: "Ambient",
	},
	{
		Title:    "Ambient: Ocean",
		URL:      "https://open.spotify.com/playlist/0czhzWKJ1qoC9iHH5yN93a",
		Category: "Ambient",
	},
	{
		Title:    "Ambient: Storm",
		URL:      "https://open.spotify.com/playlist/3lQ1VrIoMDHJmw52N3uAEc",
		Category: "Ambient",
	},
	{
		Title:    "Atmosphere: The Capital",
		URL:      "https://open.spotify.com/playlist/2t5TWAPs6HYuJ3xbpjHYpx",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Cathedral",
		URL:      "https://open.spotify.com/playlist/0IyMP3izyM2jbYgJLydB00",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Desert",
		URL:      "https://open.spotify.com/playlist/4yguXksZpqOW10hpuDyB5A",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Dungeon",
		URL:      "https://open.spotify.com/playlist/64UCYVCIPtZiOP2zEodORk",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Fey",
		URL:      "https://open.spotify.com/playlist/4jPscCOA5zrheXibHnmlU1",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Manor",
		URL:      "https://open.spotify.com/playlist/6QzZjlzHxNUo9N6E19RKpJ",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Road",
		URL:      "https://open.spotify.com/playlist/0gZQWj0PjC6t2bgmroHaaW",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Saloon",
		URL:      "https://open.spotify.com/playlist/73YmiE2tLNG5VbNF7oGmSn",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Swamp",
		URL:      "https://open.spotify.com/playlist/2xA9EIpuBH5DbmGHszQtvk",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Tavern",
		URL:      "https://open.spotify.com/playlist/2StSwZk9mV2DNO3aucMZYx",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Town",
		URL:      "https://open.spotify.com/playlist/5GgU8cLccECwAvjDCGhYjj",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Underdark",
		URL:      "https://open.spotify.com/playlist/5Qhtamj9NCxluijLnQ4edN",
		Category: "Atmosphere",
	},
	{
		Title:    "Atmosphere: The Wild",
		URL:      "https://open.spotify.com/playlist/5r2AkNQOITXRqVWqYj40QG",
		Category: "Atmosphere",
	},
	{
		Title:    "Critical Role",
		URL:      "https://open.spotify.com/playlist/5R3picMA092uzYxvvPSRGx",
		Category: "General",
	},
	{
		Title:    "PoTA: Sacred Stone Monastery",
		URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
		Category: "PoTA",
	},
	{
		Title:    "SKT: Eye of the All-Father",
		URL:      "https://open.spotify.com/playlist/3sta8W5YmT3BY2LF8sPvb1",
		Category: "SKT",
	},
	{
		Title:    "SKT: Greygate",
		URL:      "https://open.spotify.com/playlist/1c4aQPrriKV6aYVldJVFzS",
		Category: "SKT",
	},
	{
		Title:    "SKT: Maelstrom",
		URL:      "https://open.spotify.com/playlist/3dxUEDvJdWtaQWRJgKCESl",
		Category: "SKT",
	},
	{
		Title:    "Combat: Boss",
		URL:      "https://open.spotify.com/playlist/0Q6hJZYIEu3LwbyBBHjjHo",
		Category: "Combat",
	},
	{
		Title:    "Combat: Duel",
		URL:      "https://open.spotify.com/playlist/5g9ZZ9Ogml8NsjOlv8N31t",
		Category: "Combat",
	},
	{
		Title:    "Combat: Epic",
		URL:      "https://open.spotify.com/playlist/4Anyq806DQpd7pRZbSADUr",
		Category: "Combat",
	},
	{
		Title:    "Combat: Horrifying",
		URL:      "https://open.spotify.com/playlist/1SbeUQZbRHyUEIr6wsoD4q",
		Category: "Combat",
	},
	{
		Title:    "Combat: Standard",
		URL:      "https://open.spotify.com/playlist/0bWUBjlr7O4troJKyyMVbD",
		Category: "Combat",
	},
	{
		Title:    "Combat: Tough",
		URL:      "https://open.spotify.com/playlist/6T0UOAmlbWb29y2fIETtL2",
		Category: "Combat",
	},
	{
		Title:    "Feywild: Morningtide ",
		URL:      "https://open.spotify.com/playlist/60pF4EYT9L7NTjWbnpJng2",
		Category: "Feywild",
	},
	{
		Title:    "Feywild: Everbright",
		URL:      "https://open.spotify.com/playlist/34QYbrLoHRYpIBc48yMsnT",
		Category: "Feywild",
	},
	{
		Title:    "Feywild: Twilight",
		URL:      "https://open.spotify.com/playlist/3GYusL7Yx5BRfb8gn88cCR",
		Category: "Feywild",
	},
	{
		Title:    "Feywild: Everdark",
		URL:      "https://open.spotify.com/playlist/7i0RrhRpx3ALNE2ZFQrxLz",
		Category: "Feywild",
	},
	{
		Title:    "Monsters: Aberrations",
		URL:      "https://open.spotify.com/playlist/1IIfebxUOYAeOD2Aqvw7Rj",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Beasts",
		URL:      "https://open.spotify.com/playlist/6XslTVSeiQr80Gu79vnfXZ",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Dragons",
		URL:      "https://open.spotify.com/playlist/1qvLig9ELPmb8bcVPutk9M",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Giants",
		URL:      "https://open.spotify.com/playlist/6U68RdBoCkZFNWBXhQ4KXH",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Goblins",
		URL:      "https://open.spotify.com/playlist/58lGIqHs8HSmcYoKW7gBE3",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Hags",
		URL:      "https://open.spotify.com/playlist/4k1no9mrUph4rkFI1bEFJT",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Orcs",
		URL:      "https://open.spotify.com/playlist/46NfO4PokCdGvm6Fkbtx9u",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Tribesmen",
		URL:      "https://open.spotify.com/playlist/2crzs0lic8x58JyPZM8k3v",
		Category: "Monsters",
	},
	{
		Title:    "Monsters: Undead",
		URL:      "https://open.spotify.com/playlist/49PvqjRs9c4lgyvdOI4Lvd",
		Category: "Monsters",
	},
	{
		Title:    "Mood: Creepy",
		URL:      "https://open.spotify.com/playlist/6nSstCQcmzcEUSx8gBrcek",
		Category: "Mood",
	},
	{
		Title:    "Mood: Denouement",
		URL:      "https://open.spotify.com/playlist/71AETM4dyul7BDNYE9zVBv",
		Category: "Mood",
	},
	{
		Title:    "Mood: Joyful",
		URL:      "https://open.spotify.com/playlist/6KbY8nK4vdGO0zaSuoXEFr",
		Category: "Mood",
	},
	{
		Title:    "Mood: Mysterious",
		URL:      "https://open.spotify.com/playlist/28ICiQDK37yaahRZD7aX3J",
		Category: "Mood",
	},
	{
		Title:    "Mood: Ominous",
		URL:      "https://open.spotify.com/playlist/71yNeiFbb8bDhgLIzu9eae",
		Category: "Mood",
	},
	{
		Title:    "Mood: Pleasant",
		URL:      "https://open.spotify.com/playlist/3O4DGo9DS5kzUUJo6EQYdp",
		Category: "Mood",
	},
	{
		Title:    "Mood: Ridiculous",
		URL:      "https://open.spotify.com/playlist/3VepfFpcPxHIL7WyKYFdGI",
		Category: "Mood",
	},
	{
		Title:    "Mood: Serious",
		URL:      "https://open.spotify.com/playlist/3LNrO4Jvwtzk2QD1gR8ccZ",
		Category: "Mood",
	},
	{
		Title:    "Mood: Somber",
		URL:      "https://open.spotify.com/playlist/5N5w6WFXigWqZMLzVo6rdh",
		Category: "Mood",
	},
	{
		Title:    "Mood: Tense",
		URL:      "https://open.spotify.com/playlist/4DYALPIektzP4vVdZFlHNe",
		Category: "Mood",
	},
	{
		Title:    "Mood: Triumphant",
		URL:      "https://open.spotify.com/playlist/1ALzSDT8MfYQ7Xams9Nx16",
		Category: "Mood",
	},
	{
		Title:    "Setting: Barovia",
		URL:      "https://open.spotify.com/playlist/1Pw2cdOxeDBgIsocUWQYyD",
		Category: "Setting",
	},
	{
		Title:    "Setting: Chult",
		URL:      "https://open.spotify.com/playlist/4OfzULWGbFp4ohUoYuRvJh",
		Category: "Setting",
	},
	{
		Title:    "Setting: Cyberpunk",
		URL:      "https://open.spotify.com/playlist/3q2iJdKM6MqKkZoRKMtti4",
		Category: "Setting",
	},
	{
		Title:    "Setting: Film Noir",
		URL:      "https://open.spotify.com/playlist/3nn0rP52rqL4Af3GGkwtmZ",
		Category: "Setting",
	},
	{
		Title:    "Setting: Urban Fantasy",
		URL:      "https://open.spotify.com/playlist/5X5eFLCgVX4UKMZqxWFztP",
		Category: "Setting",
	},
	{
		Title:    "Situation: Chase",
		URL:      "https://open.spotify.com/playlist/1TXWTHKaWNQij6K9Ldn6fU",
		Category: "Situation",
	},
	{
		Title:    "Situation: Stealth",
		URL:      "https://open.spotify.com/playlist/6GdFG0fgrJLSXSlEkF6iM0",
		Category: "Situation",
	},
	{
		Title:    "Sea Shanties",
		URL:      "https://open.spotify.com/playlist/3p22aU2NEvY8KErZAoWSJD",
		Category: "Sea",
	},
}
