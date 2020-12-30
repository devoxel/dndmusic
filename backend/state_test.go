package main

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGuildPlaylists(t *testing.T) {
	l := newGuildPlaylists()

	testTitles := []string{"a", "b", "c", "d", "x"}
	tests := []*Playlist{}
	for _, e := range testTitles {
		pl, err := NewPlaylist(e, "cat", []Track{})
		if err != nil {
			t.Fatal(err)
		}
		tests = append(tests, pl)
	}

	if err := l.Insert(tests[0]); err != nil {
		t.Fatal(err)
	}

	if err := l.Insert(tests[0]); err == nil {
		t.Fatal(errors.New("expected error"))
	}

	if _, err := l.Get("a"); err != nil {
		t.Fatal(err)
	}

	if _, err := l.Get("b"); err == nil {
		t.Fatal(errors.New("expected error"))
	}

	if err := l.Insert(tests[len(tests)-1]); err != nil {
		t.Fatal(err)
	}

	for _, e := range tests[1 : len(tests)-1] {
		if err := l.Insert(e); err != nil {
			t.Fatal(err)
		}
	}

	got := l.GetAll()
	if diff := cmp.Diff(tests, got); diff != "" {
		t.Errorf("guildPlaylists.GetAll() mismatch (-want +got):\n%s", diff)
	}

}
