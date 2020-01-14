package kubeseal

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type ByCreationTimestamp []v1.Secret

func (s ByCreationTimestamp) Len() int {
	return len(s)
}

func (s ByCreationTimestamp) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByCreationTimestamp) Less(i, j int) bool {
	return s[i].GetCreationTimestamp().Unix() < s[j].GetCreationTimestamp().Unix()
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
