package s3utils

import (
	"fmt"
	"os"

	"github.com/rayanebel/kubeseal-backuper/pkg/config"

	"github.com/rayanebel/kubeseal-backuper/pkg/backend/s3"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

func StoreSecretKeyToS3(state *config.State, file *os.File) {
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
