package config

type Config struct {
	KubernetesClientMode        string `envconfig:"KUBERNETES_CLIENT_MODE" default:"internal"`
	KubesealControllerName      string `envconfig:"KUBESEAL_CONTROLLER_NAME" default:"kubeseal-controller"`
	KubesealControllerNamespace string `envconfig:"KUBESEAL_CONTROLLER_NAMESPACE" default:"kubeseal"`
	KubesealKeyPrefix           string `envconfig:"KUBESEAL_KEY_PREFIX" default:"sealed-secrets-key"`
	AWSBucketName               string `envconfig:"AWS_BUCKET_NAME" default:"kubeseal-key-backups" required:"true"`
	AWSRegion                   string `envconfig:"AWS_REGION" required:"true"`
	AWSAccessKey                string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	AWSSecreKey                 string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
}
