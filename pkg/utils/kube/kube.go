package k8sutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/rayanebel/kubeseal-backuper/pkg/config"
	"github.com/rayanebel/kubeseal-backuper/pkg/kube"
	"github.com/rayanebel/kubeseal-backuper/pkg/utils/kubeseal"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	kubesealSecretLabel = "sealedsecrets.bitnami.com/sealed-secrets-key"
)

// KubernetesJson2Yaml - Utils to convert k8s json into yaml
func KubernetesJson2Yaml(obj runtime.Object) (string, error) {
	tmpfile, err := ioutil.TempFile("", "kubeseal-key")
	if err != nil {
		err := fmt.Errorf("Unable to create temporary file to save yaml output: %s", err.Error())
		return "", err
	}
	defer tmpfile.Close()
	serializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory, nil, nil,
		k8sjson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		})
	err = serializer.Encode(obj, tmpfile)
	if err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

// GetGVKForObject - Retrieve Object type and version for a given object. K8s API intentionnaly remove this fields.
func GetGVKForObject(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return gvk, err
	}
	return gvk, nil
}

// SetGVKForObject - Set API version and Object type for a given k8s object.
func SetGVKForObject(obj runtime.Object) {
	gvk, _ := GetGVKForObject(obj)
	obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Version: gvk.Version,
		Group:   gvk.Group,
		Kind:    gvk.Kind,
	})
}

// CleanCommonKubernetesFields - To clean common kubernetes fields before processing.
func CleanCommonKubernetesFields(obj *unstructured.Unstructured) {
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetSelfLink("")
	obj.SetResourceVersion("")
	obj.SetUID("")
}

// SetKubernetesclient - Utils to define and init the right k8s client.
func SetKubernetesclient(state *config.State) {
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

// RestartKubesealPods - Utils to restart kubeseal pods by deleting them and let k8s recreate them.
func RestartKubesealPods(labels string, state *config.State) {
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

// CleanSecret - Utils to cleanup secret by updating custom labels and restarting kubeseal pods.
func CleanSecret(state *config.State) {

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

	RestartKubesealPods("app.kubernetes.io/instance=kubeseal", state)
	log.WithFields(log.Fields{}).Info("Kubeseal controller has been restarted.")
}
