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
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsV2Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	defaultLocation = "us-east-1"
	aesAlgo         = "AES256"
)

// generate a mock for Pathio
//go:generate bin/mockgen -source=$GOFILE -destination=gen_mock_s3handler.go -package=pathio

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
// To configure options on the client, create a new Client and call its methods
// directly.
//
//	&Client{
//		disableS3Encryption: true, // disables encryption
//		Region: "us-east-1", // hardcodes the s3 region, instead of looking it up
//	}.Write(...)
type Client struct {
	ctx                 context.Context
	disableS3Encryption bool
	Region              string
	providedConfig      *aws.Config
}

// DefaultClient is the default pathio client called by the Reader, Writer, and
// WriteReader methods. It has S3 encryption enabled.
var DefaultClient Pathio = &Client{
	ctx: context.Background(),
}

// NewClient creates a new client that utilizes the provided AWS config. This can
// be leveraged to enforce more limited permissions.
func NewClient(ctx context.Context, cfg *aws.Config) *Client {
	return &Client{
		ctx:            ctx,
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

// S3API defines the interfaces that pathio needs for AWS access.
type S3API interface {
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)

	s3.ListObjectsV2APIClient // embedded for s3's ListObjectsV2()
	s3.HeadObjectAPIClient    // embedded for s3's HeadObject()
	manager.DownloadAPIClient // embedded for s3's GetObject()

	manager.UploadAPIClient // embedded for s3's PutObject()
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// s3Handler defines the wrapper interface that pathio uses for AWS access
type s3Handler interface {
	GetBucketLocation(ctx context.Context, input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	ListObjects(ctx context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	// ListAllObjects will construct and use a ListObjectsV2 Paginator to fetch all results based on the supplied ListObjectsV2Input
	ListAllObjects(ctx context.Context, input *s3.ListObjectsV2Input) ([]*s3.ListObjectsV2Output, error)
	HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
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
		return s3FileReader(c.ctx, s3Conn)
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
	if offset, err := input.Seek(0, io.SeekStart); err != nil || offset != 0 {
		return fmt.Errorf("failed to reset the file pointer to 0. offset: %d; error %s", offset, err)
	}

	if strings.HasPrefix(path, "s3://") {
		s3Conn, err := c.s3ConnectionInformation(path, c.Region)
		if err != nil {
			return err
		}
		return writeToS3(c.ctx, s3Conn, input, c.disableS3Encryption)
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
		return deleteS3Object(c.ctx, s3Conn)
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
		return lsS3(c.ctx, s3Conn)
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
		return existsS3(c.ctx, s3Conn)
	}
	return existsLocal(path)
}

func existsS3(ctx context.Context, s3Conn s3Connection) (bool, error) {
	_, err := s3Conn.handler.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			var notFound *s3Types.NotFound
			switch {
			case errors.As(apiError, &notFound):
				return false, nil
			default:
				return false, err
			}
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

func lsS3(ctx context.Context, s3Conn s3Connection) ([]string, error) {
	params := s3.ListObjectsV2Input{
		Bucket:    aws.String(s3Conn.bucket),
		Prefix:    aws.String(s3Conn.key),
		Delimiter: aws.String("/"),
	}
	var finalResults []string

	// s3 ListObjects limits the response to 1000 objects and marks as truncated if there were more
	// In this case we set a Marker that the next query will start from.
	// We also ensure that prefixes are not duplicated
	pages, err := s3Conn.handler.ListAllObjects(ctx, &params)
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		if len(page.CommonPrefixes) > 0 && elementInSlice(finalResults, *page.CommonPrefixes[0].Prefix) {
			page.CommonPrefixes = page.CommonPrefixes[1:]
		}
		results := make([]string, len(page.Contents)+len(page.CommonPrefixes))
		for i, val := range page.CommonPrefixes {
			results[i] = *val.Prefix
		}
		for i, val := range page.Contents {
			results[i+len(page.CommonPrefixes)] = *val.Key
		}
		finalResults = append(finalResults, results...)
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
	resp, err := os.ReadDir(path)
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
func s3FileReader(ctx context.Context, s3Conn s3Connection) (io.ReadCloser, error) {
	params := s3.GetObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	}
	resp, err := s3Conn.handler.GetObject(ctx, &params)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// writeToS3 uploads the given file to S3
func writeToS3(ctx context.Context, s3Conn s3Connection, input io.ReadSeeker, disableEncryption bool) error {
	params := s3.PutObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
		Body:   input,
	}
	if !disableEncryption {
		params.ServerSideEncryption = aesAlgo
	}
	_, err := s3Conn.handler.PutObject(ctx, &params)
	return err
}

// deleteS3Object deletes the file on S3 at the given path
func deleteS3Object(ctx context.Context, s3Conn s3Connection) error {
	params := s3.DeleteObjectInput{
		Bucket: aws.String(s3Conn.bucket),
		Key:    aws.String(s3Conn.key),
	}

	_, err := s3Conn.handler.DeleteObject(ctx, &params)
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
		return "", "", fmt.Errorf("invalid s3 path %s", path)
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
		region, err = getRegionForBucket(c.ctx, c.newS3Handler(c.ctx, defaultLocation), bucket)
		if err != nil {
			return s3Connection{}, err
		}
	}

	return s3Connection{c.newS3Handler(c.ctx, region), bucket, key}, nil
}

// getRegionForBucket looks up the region name for the given bucket
func getRegionForBucket(ctx context.Context, svc s3Handler, name string) (string, error) {
	// Any region will work for the region lookup, but the request MUST use
	// PathStyle
	params := s3.GetBucketLocationInput{
		Bucket: aws.String(name),
	}
	resp, err := svc.GetBucketLocation(ctx, &params)
	if err != nil {
		return "", fmt.Errorf("failed to get location for bucket '%s', %s", name, err)
	}
	if resp.LocationConstraint == "" {
		return defaultLocation, nil
	}
	return string(resp.LocationConstraint), nil
}

type liveS3Handler struct {
	liveS3 S3API
}

func (m *liveS3Handler) GetBucketLocation(ctx context.Context, input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return m.liveS3.GetBucketLocation(ctx, input)
}

func (m *liveS3Handler) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.liveS3.GetObject(ctx, input)
}

func (m *liveS3Handler) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return m.liveS3.DeleteObject(ctx, input)
}

func (m *liveS3Handler) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return m.liveS3.PutObject(ctx, input)
}

func (m *liveS3Handler) ListObjects(ctx context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return m.liveS3.ListObjectsV2(ctx, input)
}

// ListAllObjects will utilize a ListObjectsV2Paginator to collate all responses
func (m *liveS3Handler) ListAllObjects(ctx context.Context, input *s3.ListObjectsV2Input) ([]*s3.ListObjectsV2Output, error) {
	// code reference: https://github.com/aws/aws-sdk-go-v2/blob/example/service/s3/listObjects/v0.2.9/example/service/s3/listObjects/listObjects.go
	var pages []*s3.ListObjectsV2Output
	pager := s3.NewListObjectsV2Paginator(m.liveS3, input)

	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func (m *liveS3Handler) HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return m.liveS3.HeadObject(ctx, input)
}

func (c *Client) newS3Handler(ctx context.Context, region string) *liveS3Handler {
	if c.providedConfig != nil {
		return &liveS3Handler{
			liveS3: s3.NewFromConfig(*c.providedConfig, func(o *s3.Options) {
				o.Region = region
				o.UsePathStyle = true
			}),
		}
	}

	awsConfig, err := awsV2Config.LoadDefaultConfig(ctx, awsV2Config.WithRegion(region))
	if err != nil {
		log.Fatalf("aws v2 config error: %s", err.Error())
	}

	return &liveS3Handler{s3.NewFromConfig(awsConfig)}
}
