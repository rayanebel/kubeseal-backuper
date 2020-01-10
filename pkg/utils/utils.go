package utils

import (
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
)

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func FindSecretByPrefix(secrets *v1.SecretList, prefix string) (v1.Secret, error) {
	var kubesealSecret v1.Secret

	for _, item := range secrets.Items {
		if strings.HasPrefix(item.Name, prefix) {
			kubesealSecret = item
			break
		} else {
			err := fmt.Errorf("No secret with prefix %s was found.", prefix)
			return kubesealSecret, err
		}
	}
	return kubesealSecret, nil
}
