package kube

import (
	"fmt"
	"io/ioutil"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewOutKubernetesClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func NewInKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

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

func ListSecrets(clientset *kubernetes.Clientset, namespace string, opts metav1.ListOptions) (*v1.SecretList, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(opts)
	if err != nil {
		return nil, err
	}

	if len(secrets.Items) == 0 {
		err = fmt.Errorf("No secrets with labels %s in namespace %s was found", opts.LabelSelector, namespace)
		return nil, err
	}

	return secrets, nil
}

func GetGVKForObject(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return gvk, err
	}
	return gvk, nil
}

func SetGVKForObject(obj runtime.Object) {
	gvk, _ := GetGVKForObject(obj)
	obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Version: gvk.Version,
		Group:   gvk.Group,
		Kind:    gvk.Kind,
	})
}

func CleanCommonKubernetesFields(obj *unstructured.Unstructured) {
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetSelfLink("")
	obj.SetResourceVersion("")
	obj.SetUID("")
}
