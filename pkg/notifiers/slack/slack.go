package slack

import (
	"github.com/nlopes/slack"
)

type SlackMessage struct {
	ChannelID   string
	Message     string
	Attachement slack.Attachment
}

type SlackClient struct {
	Client *slack.Client
}

// New - To init a new Slack session
func New(token string) *SlackClient {
	api := slack.New(token)
	return &SlackClient{Client: api}
}

// NewMessage - To post a message
func (s *SlackClient) NewMessage(message SlackMessage) error {
	_, _, err := s.Client.PostMessage(message.ChannelID, slack.MsgOptionText(message.Message, true), slack.MsgOptionAttachments(message.Attachement))
	if err != nil {
		return err
	}
	return nil
}
