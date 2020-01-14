package config

import (
	"github.com/aws/aws-sdk-go/aws/session"
	slackclient "github.com/rayanebel/kubeseal-backuper/pkg/notifiers/slack"
	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kube"
)

type Config struct {
	KubernetesKubeconfigPath    string `envconfig:"KUBERNETES_KUBECONFIG_PATH"`
	KubernetesClientMode        string `envconfig:"KUBERNETES_CLIENT_MODE" default:"internal"`
	KubesealControllerName      string `envconfig:"KUBESEAL_CONTROLLER_NAME" default:"kubeseal-controller"`
	KubesealControllerNamespace string `envconfig:"KUBESEAL_CONTROLLER_NAMESPACE" default:"kubeseal"`
	KubesealKeyPrefix           string `envconfig:"KUBESEAL_KEY_PREFIX" default:"sealed-secrets-key"`
	AWSBucketName               string `envconfig:"AWS_BUCKET_NAME" default:"kubeseal-key-backups" required:"true"`
	AWSRegion                   string `envconfig:"AWS_REGION" required:"true"`
	AWSAccessKey                string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	AWSSecreKey                 string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
	Notifier                    string `envconfig:"NOTIFIER" default:"slack"`
	SlackAPIToken               string `envconfig:"SLACK_API_TOKEN"`
	SlackChannelName            string `envconfig:"SLACK_CHANNEL_NAME"`
}
type State struct {
	K8s         *kube.KuberneteClient
	Config      *Config
	SlackClient *slackclient.SlackClient
	AWSClient   *session.Session
}

var state *State

func NewState() {
	state = &State{}
}

func GetState() *State {
	return state
}
