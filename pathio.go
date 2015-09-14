/*
Package pathio is a package that allows writing to and reading from different types of paths transparently.
It supports two types of paths:
 1. Local file paths
 2. S3 File Paths (s3://bucket/key)

Note that using s3 paths requires setting two environment variables
 1. AWS_SECRET_ACCESS_KEY
 2. AWS_ACCESS_KEY_ID
*/
package pathio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Reader returns an io.Reader for the specified path. The path can either be a local file path
// or an S3 path. It is the caller's responsibility to close rc.
func Reader(path string) (rc io.ReadCloser, err error) {
	if strings.HasPrefix(path, "s3://") {
		return s3FileReader(path)
	}
	// Local file path
	return os.Open(path)
}

// Write writes a byte array to the specified path. The path can be either a local file path or an
// S3 path.
func Write(path string, input []byte) error {
	return WriteReader(path, bytes.NewReader(input))
}

// WriteReader writes all the data read from the specified io.Reader to the
// output path. The path can either a local file path or an S3 path.
func WriteReader(path string, input io.ReadSeeker) error {
	if strings.HasPrefix(path, "s3://") {
		return writeToS3(path, input)
	}
	return writeToLocalFile(path, input)

}

// s3FileReader converts an S3Path into an io.ReadCloser
func s3FileReader(path string) (io.ReadCloser, error) {
	bucket, key, err := parseS3Path(path)
	if err != nil {
		return nil, err
	}

	// Look up region in S3
	region, err := getRegionForBucket(bucket)
	if err != nil {
		return nil, err
	}
	config := aws.NewConfig().WithRegion(region)

	client := s3.New(config)
	params := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	resp, err := client.GetObject(&params)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// writeToS3 uploads the given file to S3
func writeToS3(path string, input io.ReadSeeker) error {
	bucket, key, err := parseS3Path(path)
	if err != nil {
		return err
	}

	// Look up region in S3
	region, nil := getRegionForBucket(bucket)
	if err != nil {
		return err
	}
	config := aws.NewConfig().WithRegion(region)

	client := s3.New(config)
	params := s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   input,
	}
	_, err = client.PutObject(&params)
	return err
}

// writeToLocalFile writes the given file locally
func writeToLocalFile(path string, input io.ReadSeeker) error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(file, input)
	return err
}

// parseS3path parses an S3 path (s3://bucket/key) and returns a bucket, key, error tuple
func parseS3Path(path string) (string, string, error) {
	// S3 path names are of the form s3://bucket/key
	stringsArray := strings.SplitN(path, "/", 4)
	if len(stringsArray) < 4 {
		return "", "", fmt.Errorf("Invalid s3 path %s", path)
	}
	bucketName := stringsArray[2]
	// Everything after the third slash is the key
	key := stringsArray[3]
	return bucketName, key, nil
}

// getRegionForBucket looks up the region name for the given bucket
func getRegionForBucket(name string) (string, error) {
	// Any region will work for the region lookup, but the request MUST use
	// PathStyle
	config := aws.NewConfig().WithRegion("us-west-1").WithS3ForcePathStyle(true)
	client := s3.New(config)
	params := s3.GetBucketLocationInput{
		Bucket: aws.String(name),
	}
	resp, err := client.GetBucketLocation(&params)
	if err != nil {
		return "", fmt.Errorf("Failed to get location for bucket '%s', %s", name, err)
	}
	if resp.LocationConstraint == nil {
		// "US Standard", returns an empty region. So return any region in the US
		return "us-east-1", nil
	}
	return *resp.LocationConstraint, nil
}
