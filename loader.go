package config

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Loader interface {
	Load() ([]byte, error)
}

const s3URIPrefix = "s3://"

func LoaderType(uri string) string {
	if strings.HasPrefix(uri, s3URIPrefix) {
		return "s3"
	}
	return "file"
}

type FileConfig struct {
	Path string
}
type FileLoader struct {
	config FileConfig
}

func NewFileLoader(rawConfig interface{}) (*FileLoader, error) {
	if config, ok := rawConfig.(FileConfig); ok {
		return &FileLoader{config: config}, nil
	}
	return nil, errors.New("config must be of type `FileConfig`")
}

// Load grabs configuration from a file
func (l *FileLoader) Load() ([]byte, error) {

	if !pathExists(l.config.Path) {
		return nil, errors.New("invalid file path")
	}

	return ioutil.ReadFile(l.config.Path)

}

// pathExists checks if an os file path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	// IsNotExist() will return false if err == nil
	// For a full explanation of IsNotExist(), see
	// https://golang.org/pkg/os/#IsNotExist
	return !os.IsNotExist(err)
}

// S3ConfigFromURI parses a URI string into an S3Config
// s3://BUCKET/OBJECT
func S3ConfigFromURI(uri string) (*S3Config, error) {
	if uri[0:5] != s3URIPrefix {
		return nil, errors.New("uri not of format s3://<region>/<bucket>/<key>")
	}
	uri = strings.TrimPrefix(uri, s3URIPrefix)

	uriParts := strings.SplitN(uri, "/", 2)
	if len(uriParts) < 3 {
		return nil, errors.New("uri not of format s3://<region>/<bucket>/<key>")
	}
	return &S3Config{
		Region: uriParts[0],
		Bucket: uriParts[1],
		Key:    uriParts[2],
	}, nil
}

type S3Config struct {
	Region string
	Bucket string
	Key    string
}
type S3Loader struct {
	config S3Config
}

func NewS3Loader(rawConfig interface{}) (*S3Loader, error) {
	if config, ok := rawConfig.(S3Config); ok {
		return &S3Loader{config: config}, nil
	}
	return nil, errors.New("config must be of type `S3Config`")
}

// Load grabs configuration from s3. This will use whatever credentials
// you have in your environment
func (l *S3Loader) Load() ([]byte, error) {

	client := s3.New(session.New(), &aws.Config{Region: aws.String(l.config.Region)})

	// ensure the desired s3 bucket exists and is accessible
	_, err := client.GetBucketVersioning(
		&s3.GetBucketVersioningInput{
			Bucket: aws.String(l.config.Bucket),
		},
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String(l.config.Bucket),
			Key:    aws.String(l.config.Key),
		},
	)
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if reqErr.StatusCode() == 404 {
				return nil, errors.New("s3 config not found")
			}
		}
		return nil, err
	}

	conf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return conf, nil

}
