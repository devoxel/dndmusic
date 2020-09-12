package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type mockDiscordSession struct {
	gotChannelID string
	gotContent   string

	returnsMessage *discordgo.Message
	returnsError   error
	called         int
}

func (ds *mockDiscordSession) ChannelMessageSend(channelID string, content string) (*discordgo.Message, error) {
	ds.called += 1
	ds.gotChannelID = channelID
	ds.gotContent = content

	return ds.returnsMessage, ds.returnsError
}

func TestIncomingMessageHandler(t *testing.T) {
	testSession := &Sessions{
		states:       map[string]*guildState{},
		pwValidation: map[string]string{},
	}
	pw := testSession.Password("anyid")
	pw2 := testSession.Password("anyid2")

	tests := []struct {
		session        *Sessions
		mockSession    *mockDiscordSession
		messageContent string

		wantNotCalled bool
		wantResponse  string
	}{
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "🙂 create " + pw,
			wantResponse:   "good to go",
		},
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "🙂 CREATE " + pw2,
			wantResponse:   "good to go",
		},
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "🙂 create badpassword",
			wantResponse:   "invalid password",
		},
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "🙂 create",
			wantResponse:   "enter a password",
		},
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "🙂",
			wantResponse:   "you're using the bot wrong.",
		},
		{
			session:        testSession,
			mockSession:    &mockDiscordSession{},
			messageContent: "unrelated discord messgae",
			wantNotCalled:  true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			ds := DiscordServer{test.session}
			mc := &discordgo.MessageCreate{&discordgo.Message{
				Content: test.messageContent,
			}}

			ds.incomingMessage(test.mockSession, mc)
			if test.wantNotCalled && test.mockSession.called == 0 {
				return
			} else if test.wantNotCalled {
				t.Fatal("ChannelMessageSend called when we want no calls")
			}

			if test.mockSession.called == 0 {
				t.Fatal("ChannelMessageSend not called")
			}

			got := test.mockSession.gotContent
			want := test.wantResponse
			if !strings.Contains(got, want) {
				t.Fatalf("ChannelMessageSend('%s') does not contain %s", got, want)
			}

		})

	}

}
