package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func New(region string) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func PutObject(session *session.Session, input *s3manager.UploadInput) error {
	uploader := s3manager.NewUploader(session)
	_, err := uploader.Upload(input)
	if err != nil {
		return err
	}
	return nil
}
