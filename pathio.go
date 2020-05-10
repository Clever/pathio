// Package pathio is a package that allows writing to and reading from different types of paths transparently.
// It supports two types of paths:
//  1. Local file paths
//  2. S3 File Paths (s3://bucket/key)
//
// Note that using s3 paths requires setting two environment variables
//  1. AWS_SECRET_ACCESS_KEY
//  2. AWS_ACCESS_KEY_ID
package pathio

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	defaultLocation = "us-east-1"
	aesAlgo         = "AES256"
)

// generate a mock for Pathio
//go:generate $GOPATH/bin/mockgen -source=$GOFILE -destination=gen_mock_s3handler.go -package=pathio

// Pathio is a defined interface for accessing both S3 and local files.
type Pathio interface {
	Reader(path string) (rc io.ReadCloser, err error)
	Write(path string, input []byte) error
	WriteReader(path string, input io.ReadSeeker) error
	Delete(path string) error
	ListFiles(path string) ([]string, error)
	Exists(path string) (bool, error)
}

// Client is the pathio client used to access the local file system and S3.
type Client struct {
	Region         string
	providedConfig *aws.Config
}

// DefaultClient is the default pathio client called by the Reader, Writer, and
// WriteReader methods. It has S3 encryption enabled.
var DefaultClient Pathio = &Client{}

// NewClient creates a new client that utilizes the provided AWS config. This can
// be leveraged to enforce more limited permissions.
func NewClient(cfg *aws.Config) *Client {
	return &Client{
		providedConfig: cfg,
	}
}

// Reader calls DefaultClient's Reader method.
func Reader(path string) (rc io.ReadCloser, err error) {
	return DefaultClient.Reader(path)
}

// Write calls DefaultClient's Write method.
func Write(path string, input []byte) error {
	return DefaultClient.Write(path, input)
}

// WriteReader calls DefaultClient's WriteReader method.
func WriteReader(path string, input io.ReadSeeker) error {
	return DefaultClient.WriteReader(path, input)
}

// Delete calls DefaultClient's Delete method.
func Delete(path string) error {
	return DefaultClient.Delete(path)
}

// ListFiles calls DefaultClient's ListFiles method.
func ListFiles(path string) ([]string, error) {
	return DefaultClient.ListFiles(path)
}

// Exists calls DefaultClient's Exists method.
func Exists(path string) (bool, error) {
	return DefaultClient.Exists(path)
}

// s3Handler defines the interface that pathio needs for AWS access.
type s3Handler interface {
	GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
}

type s3Connection struct {
	handler s3Handler
	bucket  string
	key     string
}

// Reader returns an io.Reader for the specified path. The path can either be a local file path
// or an S3 path. It is the caller's responsibility to close rc.
func (c *Client) Reader(path string) (rc io.ReadCloser, err error) {
	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return nil, err
		}
		return s3FileReader(s3Conn)
	}
	// Local file path
	return os.Open(path)
}

// Write writes a byte array to the specified path. The path can be either a local file path or an
// S3 path.
func (c *Client) Write(path string, input []byte) error {
	return c.WriteReader(path, bytes.NewReader(input))
}

// WriteReader writes all the data read from the specified io.Reader to the
// output path. The path can either a local file path or an S3 path.
func (c *Client) WriteReader(path string, input io.ReadSeeker) error {
	// return the file pointer to the start before reading from it when writing
	if offset, err := input.Seek(0, os.SEEK_SET); err != nil || offset != 0 {
		return fmt.Errorf("failed to reset the file pointer to 0. offset: %d; error %s", offset, err)
	}

	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return err
		}
		return writeToS3(s3Conn, input)
	}
	return writeToLocalFile(path, input)
}

// Delete deletes the object at the specified path. The path can be either
// a local file path or an S3 path.
func (c *Client) Delete(path string) error {
	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return err
		}
		return deleteS3Object(s3Conn)
	}
	// Local file path
	return os.Remove(path)
}

// ListFiles lists all the files/directories in the directory. It does not recurse
func (c *Client) ListFiles(path string) ([]string, error) {
	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return nil, err
		}
		return lsS3(s3Conn)
	}
	return lsLocal(path)
}

// Exists determines if a path does or does not exist.
// NOTE: S3 is eventually consistent so keep in mind that there is a delay.
func (c *Client) Exists(path string) (bool, error) {
	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return false, err
		}
		return existsS3(s3Conn)
	}
	return existsLocal(path)
}

func existsS3(s3Conn s3Connection) (bool, error) {
	_, err := s3Conn.handler.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	})
	if err != nil {
		if aerr, ok := err.(s3.RequestFailure); ok && aerr.StatusCode() == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func existsLocal(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func lsS3(s3Conn s3Connection) ([]string, error) {
	params := s3.ListObjectsInput{
		Bucket:    aws.String(s3Conn.bucket),
		Prefix:    aws.String(s3Conn.key),
		Delimiter: aws.String("/"),
	}
	finalResults := []string{}

	// s3 ListObjects limits the respose to 1000 objects and marks as truncated if there were more
	// In this case we set a Marker that the next query will start from.
	// We also ensure that prefixes are not duplicated
	for {
		resp, err := s3Conn.handler.ListObjects(&params)
		if err != nil {
			return nil, err
		}
		if len(resp.CommonPrefixes) > 0 && elementInSlice(finalResults, *resp.CommonPrefixes[0].Prefix) {
			resp.CommonPrefixes = resp.CommonPrefixes[1:]
		}
		results := make([]string, len(resp.Contents)+len(resp.CommonPrefixes))
		for i, val := range resp.CommonPrefixes {
			results[i] = *val.Prefix
		}
		for i, val := range resp.Contents {
			results[i+len(resp.CommonPrefixes)] = *val.Key
		}
		finalResults = append(finalResults, results...)

		if resp.IsTruncated != nil && *resp.IsTruncated {
			params.Marker = aws.String(results[len(results)-1])
		} else {
			break
		}
	}
	return finalResults, nil
}

func elementInSlice(slice []string, elem string) bool {
	for _, v := range slice {
		if elem == v {
			return true
		}
	}
	return false
}

func lsLocal(path string) ([]string, error) {
	resp, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(resp))
	for i, val := range resp {
		results[i] = val.Name()
	}
	return results, nil
}

// s3FileReader converts an S3Path into an io.ReadCloser
func s3FileReader(s3Conn s3Connection) (io.ReadCloser, error) {
	params := s3.GetObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	}
	resp, err := s3Conn.handler.GetObject(&params)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// writeToS3 uploads the given file to S3
func writeToS3(s3Conn s3Connection, input io.ReadSeeker) error {
	params := s3.PutObjectInput{
		Bucket:               aws.String(s3Conn.bucket),
		Key:                  aws.String(s3Conn.key),
		Body:                 input,
		ServerSideEncryption: aws.String(aesAlgo),
		StorageClass:         aws.String(s3.StorageClassIntelligentTiering),
	}
	_, err := s3Conn.handler.PutObject(&params)
	return err
}

// deleteS3Object deletes the file on S3 at the given path
func deleteS3Object(s3Conn s3Connection) error {
	params := s3.DeleteObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	}

	_, err := s3Conn.handler.DeleteObject(&params)
	return err
}

// writeToLocalFile writes the given file locally
func writeToLocalFile(path string, input io.ReadSeeker) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
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

// s3ConnectionInformation parses the s3 path and returns the s3 connection from the
// correct region, as well as the bucket, and key
func (c *Client) s3ConnectionInformation(path, region string) (s3Connection, error) {
	bucket, key, err := parseS3Path(path)
	if err != nil {
		return s3Connection{}, err
	}

	// If no region passed in, look up region in S3
	if region == "" {
		region, err = getRegionForBucket(c.newS3Handler(defaultLocation), bucket)
		if err != nil {
			return s3Connection{}, err
		}
	}

	return s3Connection{c.newS3Handler(region), bucket, key}, nil
}

// getRegionForBucket looks up the region name for the given bucket
func getRegionForBucket(svc s3Handler, name string) (string, error) {
	// Any region will work for the region lookup, but the request MUST use
	// PathStyle
	params := s3.GetBucketLocationInput{
		Bucket: aws.String(name),
	}
	resp, err := svc.GetBucketLocation(&params)
	if err != nil {
		return "", fmt.Errorf("Failed to get location for bucket '%s', %s", name, err)
	}
	if resp.LocationConstraint == nil {
		// "US Standard", returns an empty region, which means us-east-1
		// See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTBucketGETlocation.html
		return defaultLocation, nil
	}
	return *resp.LocationConstraint, nil
}

type liveS3Handler struct {
	liveS3 *s3.S3
}

func (m *liveS3Handler) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return m.liveS3.GetBucketLocation(input)
}

func (m *liveS3Handler) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.liveS3.GetObject(input)
}

func (m *liveS3Handler) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return m.liveS3.DeleteObject(input)
}

func (m *liveS3Handler) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return m.liveS3.PutObject(input)
}

func (m *liveS3Handler) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	return m.liveS3.ListObjects(input)
}

func (m *liveS3Handler) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return m.liveS3.HeadObject(input)
}

func (c *Client) newS3Handler(region string) *liveS3Handler {
	if c.providedConfig != nil {
		return &liveS3Handler{
			liveS3: s3.New(session.New(), c.providedConfig.WithRegion(region).WithS3ForcePathStyle(true)),
		}
	}

	config := aws.NewConfig().WithRegion(region).WithS3ForcePathStyle(true)
	session := session.New()
	return &liveS3Handler{s3.New(session, config)}
}
