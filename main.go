package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nlopes/slack"
	"github.com/rayanebel/kubeseal-backuper/pkg/config"

	slackclient "github.com/rayanebel/kubeseal-backuper/pkg/notifiers/slack"

	k8sutils "github.com/rayanebel/kubeseal-backuper/pkg/utils/kube"
	s3utils "github.com/rayanebel/kubeseal-backuper/pkg/utils/s3"
	slackutils "github.com/rayanebel/kubeseal-backuper/pkg/utils/slack"

	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kubeseal"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	kubesealSecretLabel = "sealedsecrets.bitnami.com/sealed-secrets-key"
)

var state *config.State

func run() {
	k8sutils.SetKubernetesclient(state)
	processBackup(state)
	k8sutils.CleanSecret(state)

	msgTxt := fmt.Sprintf("*Kubeseal controller*: `%s` has generated a new encryption key."+
		" This Key has been upload to S3."+
		" The old encryption keys have all been *decommissioned*."+
		" Please *re-encrypt* all your secret using the new key.", state.Config.KubesealControllerName)
	switch state.Config.Notifier {
	case "slack":
		slackMsg := slackclient.SlackMessage{
			Message:   "",
			ChannelID: state.Config.SlackChannelName,
			Attachement: slack.Attachment{
				Title: ":robot_face: Kubeseal Operator",
				Color: "#00FF00",
				Text:  msgTxt,
			}}
		slackutils.InitSlack(state)
		slackutils.NotifySlack(state, slackMsg)
	default:
		log.WithFields(log.Fields{
			"notifier": state.Config.Notifier,
		}).Error("Unsupported notifier backend")
		os.Exit(1)
	}
}

func processBackup(state *config.State) {
	labelSelector := kubesealSecretLabel
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	secrets, err := state.K8s.ListSecrets(state.Config.KubesealControllerNamespace, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err.Error(),
			"namespace": state.Config.KubesealControllerNamespace,
		}).Error("Unable to list secrets")
		os.Exit(1)
	}

	secret, err := kubeseal.FindSecretByPrefix(secrets, state.Config.KubesealKeyPrefix)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to find kubeseal secret")
		os.Exit(1)
	}

	k8sutils.SetGVKForObject(&secret)

	var obj unstructured.Unstructured
	objByte, err := json.Marshal(secret)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Serialization error")
		os.Exit(1)
	}

	err = json.Unmarshal(objByte, &obj)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Deserialization error")
		os.Exit(1)
	}

	k8sutils.CleanCommonKubernetesFields(&obj)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &secret)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to convert unstructured object into v1.Secret")
		os.Exit(1)
	}

	fileName, err := k8sutils.KubernetesJson2Yaml(&secret)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to convert secret object into yaml")
		os.Exit(1)
	}

	kubesealYamlfile, err := os.Open(fileName)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"filename": fileName,
		}).Error("Unable to open temporary file")
		os.Exit(1)
	}
	defer kubesealYamlfile.Close()

	s3utils.StoreSecretKeyToS3(state, kubesealYamlfile)

}

func main() {
	conf := &config.Config{}
	err := envconfig.Process("", conf)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Config error")
		os.Exit(1)
	}
	config.NewState()
	state = config.GetState()
	state.Config = conf
	run()
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}
