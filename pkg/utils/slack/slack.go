package slackutils

import (
	"os"

	"github.com/rayanebel/kubeseal-backuper/pkg/config"

	slackclient "github.com/rayanebel/kubeseal-backuper/pkg/notifiers/slack"

	log "github.com/sirupsen/logrus"
)

// InitSlack - Utils to check and init slack client.
func InitSlack(state *config.State) {
	if state.Config.SlackAPIToken == "" {
		log.Error("Config error: mission Slack API Token")
		os.Exit(1)
	}
	if state.Config.SlackChannelName == "" {
		log.Error("Config error: mission Slack Channel ID")
		os.Exit(1)
	}
	state.SlackClient = slackclient.New(state.Config.SlackAPIToken)
}

// NotifySlack - Utils to send message to slack.
func NotifySlack(state *config.State, message slackclient.SlackMessage) {
	err := state.SlackClient.NewMessage(message)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err.Error(),
			"channel": state.Config.SlackAPIToken,
		}).Error("Unable to post message to slack")
		os.Exit(1)
	}

}
