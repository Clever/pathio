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
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const defaultLocation = "us-east-1"
const aesAlgo = "AES256"

// Client is the pathio client used to access the local file system and S3. It
// includes an option to disable S3 encryption. To disable S3 encryption, create
// a new Client and call it directly:
// `&Client{disableS3Encryption: true}.Write(...)`
type Client struct {
	disableS3Encryption bool
}

// DefaultClient is the default pathio client called by the Reader, Writer, and
// WriteReader methods. It has S3 encryption enabled.
var DefaultClient = &Client{}

// Reader Calls DefaultClient's Reader method.
func Reader(path string) (rc io.ReadCloser, err error) {
	return DefaultClient.Reader(path)
}

// Write Calls DefaultClient's Write method.
func Write(path string, input []byte) error {
	return DefaultClient.Write(path, input)
}

// Stream allows you to stream data directly to file via a reader.
//
// WARNING 1: Being that this function is async and somewhat tricky,
// only use if it you know exactly what's happening.
//
// WARNING 2: The go routine does *not* return until the input reader is closed.
//
// The intended use case for this function is to create a direct stream to a path
// via an io.Pipe() reader/writer. Everytime the writer writes, the reader has something to read.
// Note, for an io.Pipe(), once you close the writer you also close the reader, they're
// entagled objects.
//
// Example:
//
// func main() {
// 	reader, writer := io.Pipe()
// 	path := "some path"
// 	var err *error
// 	pathio.Stream(path, reader, err)
// 	// generate stuff
// 	writer.Write(stuff)
// 	writer.Close()
// 	if err != nil {
// 		// Handle Error
// 	}
// }
//
// Note on error handling: Because the function is a Go routine, we implicitly
// handle errors by having an error pointer passed in.
func Stream(path string, input io.Reader, err *error) {
	if strings.HasPrefix(path, "s3://") {
		streamToS3(path, input, err)
	} else {
		streamToLocalFile(path, input, err)
	}
}

// streamToS3 establishes an open stream to an s3 File
func streamToS3(path string, input io.Reader, err *error) {
	go func() {
		s3Conn, connErr := s3ConnectionInformation(path)
		if connErr != nil {
			err = &connErr
			return
		}
		upParams := &s3manager.UploadInput{
			Bucket: aws.String(s3Conn.bucket),
			Key:    aws.String(s3Conn.key),
			Body:   input,
		}

		region, regionErr := getRegionForBucket(newS3Handler(defaultLocation), s3Conn.bucket)
		if regionErr != nil {
			err = &regionErr
			return
		}
		config := aws.NewConfig().WithRegion(region).WithS3ForcePathStyle(true)
		client := s3.New(session.New(), config)
		uploader := s3manager.NewUploaderWithClient(client)
		_, uploadErr := uploader.Upload(upParams)
		if uploadErr != nil {
			err = &uploadErr
		}
		return
	}()
}

// streamToLocalFile establishes an open stream to a local file
func streamToLocalFile(path string, input io.Reader, err *error) {
	go func() {
		// Open a file, if it doesn't exist create it.
		var file *os.File
		file, fileErr := os.Create(path)
		if fileErr != nil {
			err = &fileErr
			return
		}
		_, copyErr := io.Copy(file, input)
		if copyErr != nil {
			err = &copyErr
		}
		return
	}()
}

// WriteReader Calls DefaultClient's WriteReader method.
func WriteReader(path string, input io.ReadSeeker) error {
	return DefaultClient.WriteReader(path, input)
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
		s3Conn, err := s3ConnectionInformation(path)
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
	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := s3ConnectionInformation(path)
		if err != nil {
			return err
		}
		return writeToS3(s3Conn, input, c.disableS3Encryption)
	}
	return writeToLocalFile(path, input)
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
func writeToS3(s3Conn s3Connection, input io.ReadSeeker, disableEncryption bool) error {
	params := s3.PutObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
		Body:   input,
	}
	if !disableEncryption {
		algo := aesAlgo
		params.ServerSideEncryption = &algo
	}
	_, err := s3Conn.handler.PutObject(&params)
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
func s3ConnectionInformation(path string) (s3Connection, error) {
	bucket, key, err := parseS3Path(path)
	if err != nil {
		return s3Connection{}, err
	}

	// Look up region in S3
	region, err := getRegionForBucket(newS3Handler(defaultLocation), bucket)
	if err != nil {
		return s3Connection{}, err
	}

	return s3Connection{newS3Handler(region), bucket, key}, nil
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

type s3Handler interface {
	GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
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

func (m *liveS3Handler) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return m.liveS3.PutObject(input)
}

func newS3Handler(region string) *liveS3Handler {
	config := aws.NewConfig().WithRegion(region).WithS3ForcePathStyle(true)
	session := session.New()
	return &liveS3Handler{s3.New(session, config)}
}
