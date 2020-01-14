package kube

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KuberneteClient struct {
	Client *kubernetes.Clientset
}

// NewOutKubernetesClient - To init an external k8s client
func NewOutKubernetesClient(kubeConfigPath string) (*KuberneteClient, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return &KuberneteClient{Client: clientset}, nil
}

// NewInKubernetesClient - To init an internal k8s client
func NewInKubernetesClient() (*KuberneteClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KuberneteClient{Client: clientset}, nil
}

// ListSecrets - To list all k8s secrets
func (s *KuberneteClient) ListSecrets(namespace string, opts metav1.ListOptions) (*v1.SecretList, error) {
	secrets, err := s.Client.CoreV1().Secrets(namespace).List(opts)
	if err != nil {
		return nil, err
	}

	if len(secrets.Items) == 0 {
		err = fmt.Errorf("No secrets with labels %s in namespace %s was found", opts.LabelSelector, namespace)
		return nil, err
	}

	return secrets, nil
}

// ListPods - To list all k8s pods
func (s *KuberneteClient) ListPods(namespace string, opts metav1.ListOptions) (*v1.PodList, error) {
	pods, err := s.Client.CoreV1().Pods(namespace).List(opts)
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		err = fmt.Errorf("No secrets with labels %s in namespace %s was found", opts.LabelSelector, namespace)
		return nil, err
	}

	return pods, nil
}

// UpdateSecret - To update a given secret
func (s *KuberneteClient) UpdateSecret(updatedSecret *v1.Secret) error {
	_, err := s.Client.CoreV1().Secrets(updatedSecret.Namespace).Update(updatedSecret)
	if err != nil {
		return fmt.Errorf("Unable to update secrets: %s", err.Error())
	}
	return nil
}

// DeletePods - To delete a list of pods
func (s *KuberneteClient) DeletePods(pods *v1.PodList) error {
	for _, pod := range pods.Items {
		log.WithFields(log.Fields{
			"podName": pod.Name,
		}).Warning("Trying to delete pod")
		err := s.Client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			msg := fmt.Errorf("Unable to delete pod %s: ", err.Error())
			return msg
		}
		log.WithFields(log.Fields{
			"podName": pod.Name,
		}).Info("Pod has been deleted")
	}
	return nil

}
