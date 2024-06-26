package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
)

// prefix is basically an optional string which gets prepended to name of the file
func UploadToBucket(file *multipart.FileHeader, prefix string) error {
	err := godotenv.Load()
	if err != nil {
		return errors.New(fmt.Sprintf("Error loading environment variables: %s", err.Error()))
	}

	accessKey := os.Getenv("BUCKET_ACCESS_KEY")
	secretKey := os.Getenv("BUCKET_SECRET_KEY")
	bucketName := os.Getenv("BUCKET_NAME")
	endpoint := os.Getenv("BUCKET_ENDPOINT")

	if accessKey == "" || secretKey == "" || bucketName == "" {
		return errors.New(fmt.Sprintf("Environment variables are not loaded correctly"))
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating session: %s", err.Error()))
	}

	client := s3.New(sess)

	f, err := file.Open()
	defer f.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("Error opening file: %s", err.Error()))
	}

	// Read the contents of the file into a buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return errors.New(fmt.Sprintf("Error reading file: %s", err.Error()))
	}

	destinationKey := strings.Join([]string{
		"cloudbuddy/",
		prefix,
		"-",
		file.Filename,
	}, "")

	// This uploads the contents of the buffer to S3
	_, err = client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(destinationKey),
		Body:   bytes.NewReader(buf.Bytes()),
	})

	if err != nil {
		return errors.New(fmt.Sprintf("Error uploading file: %s", err.Error()))
	}

	return nil
}
