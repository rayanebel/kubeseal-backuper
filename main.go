package main

import (
	"encoding/json"
	"fmt"
	"github.com/rayanebel/kubeseal-backuper/pkg/config"
	"log"
	"os"
	"path/filepath"

	"github.com/rayanebel/kubeseal-backuper/pkg/backend/s3"
	"github.com/rayanebel/kubeseal-backuper/pkg/utils"
	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kube"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

const (
	kubesealSecretLabel = "sealedsecrets.bitnami.com/sealed-secrets-key"
)

func setKubernetesclient(mode string) *kubernetes.Clientset {
	var clientset *kubernetes.Clientset
	var err error
	switch mode {
	case "external":
		home := utils.HomeDir()
		kubeconfigPath := filepath.Join(home, ".kube", "config")
		clientset, err = kube.NewOutKubernetesClient(kubeconfigPath)

		if err != nil {
			log.Fatalf("Unable to init external kubernetes client: %s", err.Error())
		}
	case "internal":
		clientset, err = kube.NewInKubernetesClient()

		if err != nil {
			log.Fatalf("Unable to init internal kubernetes client: %s", err.Error())
		}
	}
	return clientset
}

func runBackup(config *config.Config) {
	clientset := setKubernetesclient(config.KubernetesClientMode)

	labelSelector := kubesealSecretLabel
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	secrets, err := kube.ListSecrets(clientset, config.KubesealControllerNamespace, opts)
	if err != nil {
		log.Fatalf("Unable to list secrets in namespace %s: %s", config.KubesealControllerNamespace, err.Error())
	}

	secret, err := utils.FindSecretByPrefix(secrets, config.KubesealKeyPrefix)
	if err != nil {
		log.Fatalf("Unable to find kubeseal secret: %s", err.Error())
	}

	kube.SetGVKForObject(&secret)

	var obj unstructured.Unstructured
	objByte, err := json.Marshal(secret)
	if err != nil {
		log.Fatalf("Serialization error: %s", err.Error())
	}

	err = json.Unmarshal(objByte, &obj)
	if err != nil {
		log.Fatalf("Deserialization error: %s", err.Error())
	}

	kube.CleanCommonKubernetesFields(&obj)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &secret)
	if err != nil {
		log.Fatalf("Unable to convert unstructured object into v1.Secret: %s", err.Error())
	}

	fileName, err := kube.KubernetesJson2Yaml(&secret)
	if err != nil {
		log.Fatalf("Unable to convert secret object into yaml: %s", err.Error())
	}

	kubesealYamlfile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Unable to open temporary file %s: %s", fileName, err.Error())
	}
	defer kubesealYamlfile.Close()

	session, err := s3.New(config.AWSRegion)
	if err != nil {
		log.Fatalf("Unable to open session to AWS: %s", err.Error())
	}

	keyName := fmt.Sprintf("%s/%s-key.yaml", config.KubesealControllerNamespace, config.KubesealControllerName)
	payload := &s3manager.UploadInput{
		Bucket: &config.AWSBucketName,
		Key:    &keyName,
		Body:   kubesealYamlfile,
	}
	err = s3.PutObject(session, payload)
	if err != nil {
		log.Fatalf("Unable to upload kubeseal key in the bucket %s: %s", config.AWSBucketName, err.Error())
	}
	fmt.Printf("New file: %s has been upload to s3: %s", keyName, config.AWSBucketName)
}

func main() {
	config := config.Config{}
	err := envconfig.Process("", &config)

	if err != nil {
		log.Fatalf("Config error: %s", err.Error())
	}
	runBackup(&config)
}
