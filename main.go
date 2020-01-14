package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/nlopes/slack"
	"github.com/rayanebel/kubeseal-backuper/pkg/config"

	"github.com/rayanebel/kubeseal-backuper/pkg/backend/s3"
	slackclient "github.com/rayanebel/kubeseal-backuper/pkg/notifiers/slack"

	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kube"
	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kubeseal"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

func setKubernetesclient(state *config.State) {
	log.WithFields(log.Fields{
		"mode": state.Config.KubernetesClientMode,
	}).Info("Trying to setup kubernetes client")
	var err error
	switch state.Config.KubernetesClientMode {
	case "external":
		if state.Config.KubernetesKubeconfigPath == "" {
			log.WithFields(log.Fields{}).Error("No kubeconfig path has been provided. Please set KUBERNETES_KUBECONFIG_PATH if your are in external mode")
			os.Exit(1)
		}
		state.K8s, err = kube.NewOutKubernetesClient(state.Config.KubernetesKubeconfigPath)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Unable to init external kubernetes client")
			os.Exit(1)
		}
	case "internal":
		state.K8s, err = kube.NewInKubernetesClient()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Unable to init internal kubernetes client")
			os.Exit(1)
		}
	default:
		log.WithFields(log.Fields{}).Error("Unable to init kubernetes client: client mode set is invalid.")
		os.Exit(1)
	}
}

//app.kubernetes.io/instance=kubeseal
func restartKubesealPods(labels string, state *config.State) {
	// Add controller-name as labels to use config.KubesealControllerName
	opts := metav1.ListOptions{
		LabelSelector: labels,
	}
	kubesealPods, err := state.K8s.ListPods(state.Config.KubesealControllerNamespace, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err.Error(),
			"namespace": state.Config.KubesealControllerNamespace,
		}).Error("Unable to list pods")
		os.Exit(1)
	}
	err = state.K8s.DeletePods(kubesealPods)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to delete kubeseal pods")
		os.Exit(1)
	}
}

// cleanup secret by setting old key as compromised and restart pods
func cleanSecret(state *config.State) {

	labelSelector := kubesealSecretLabel
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	list, _ := state.K8s.ListSecrets(state.Config.KubesealControllerNamespace, opts)
	sort.Sort(kubeseal.ByCreationTimestamp(list.Items))
	latestKey := &list.Items[len(list.Items)-1]

	log.WithFields(log.Fields{
		"latest": latestKey.Name,
	}).Info("Latest sealed secret")

	for _, key := range list.Items {
		if key.Name == latestKey.Name {
			continue
		}
		log.WithFields(log.Fields{
			"key": key.Name,
		}).Info("Disable secret key")

		key.Labels[kubesealSecretLabel] = "compromised"
		state.K8s.UpdateSecret(&key)
	}
	log.WithFields(log.Fields{
		"labels": "app.kubernetes.io/instance=kubeseal",
	}).Warning("Restarting kubeseal controller with labels")

	restartKubesealPods("app.kubernetes.io/instance=kubeseal", state)
	log.WithFields(log.Fields{}).Info("Kubeseal controller has been restarted.")
}

func initSlack(state *config.State) {
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

func notifySlack(state *config.State, message slackclient.SlackMessage) {
	err := state.SlackClient.NewMessage(message)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err.Error(),
			"channel": state.Config.SlackAPIToken,
		}).Error("Unable to post message to slack")
		os.Exit(1)
	}
}

func storeSecretKeyToS3(state *config.State, file *os.File) {
	var err error
	state.AWSClient, err = s3.New(state.Config.AWSRegion)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to open session to AWS")
		os.Exit(1)
	}
	keyName := fmt.Sprintf("%s/%s-key.yaml", state.Config.KubesealControllerNamespace, state.Config.KubesealControllerName)
	payload := &s3manager.UploadInput{
		Bucket: &state.Config.AWSBucketName,
		Key:    &keyName,
		Body:   file,
	}
	err = s3.PutObject(state.AWSClient, payload)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err.Error(),
			"bucket": state.Config.AWSBucketName,
		}).Error("Unable to upload kubeseal key in the bucket configured")
		os.Exit(1)
	}
	log.WithFields(log.Fields{
		"filename": keyName,
		"bucket":   state.Config.AWSBucketName,
	}).Info("New key file has been upload to s3")
}

func run() {
	setKubernetesclient(state)
	processBackup(state)
	cleanSecret(state)

	msgTxt := fmt.Sprintf("*Kubeseal controller*: `%s` has generated a new encryption key."+
		" This Key has been upload to S3."+
		" The old encryption keys have all been *decommissioned*."+
		" Please *re-encrypt* all your secret using the new key.", state.Config.KubesealControllerName)
	switch state.Config.Notifier {
	case "slack":
		slackMsg := slackclient.SlackMessage{
			Message:   "",
			ChannelID: "testbot",
			Attachement: slack.Attachment{
				Title: ":robot_face: Kubeseal Operator",
				Color: "#00FF00",
				Text:  msgTxt,
			}}
		initSlack(state)
		notifySlack(state, slackMsg)
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

	kube.SetGVKForObject(&secret)

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

	kube.CleanCommonKubernetesFields(&obj)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &secret)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to convert unstructured object into v1.Secret")
		os.Exit(1)
	}

	fileName, err := kube.KubernetesJson2Yaml(&secret)
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

	storeSecretKeyToS3(state, kubesealYamlfile)

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
