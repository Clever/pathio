package pathio

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestParseS3Path(t *testing.T) {
	bucketName, s3path, err := parseS3Path("s3://clever-files/directory/path")
	assert.Nil(t, err)
	assert.Equal(t, bucketName, "clever-files")
	assert.Equal(t, s3path, "directory/path")

	bucketName, s3path, err = parseS3Path("s3://clever-files/directory")
	assert.Nil(t, err)
	assert.Equal(t, bucketName, "clever-files")
	assert.Equal(t, s3path, "directory")
}

func TestParseInvalidS3Path(t *testing.T) {
	_, _, err := parseS3Path("s3://")
	assert.EqualError(t, err, "invalid s3 path s3://")

	_, _, err = parseS3Path("s3://ag-ge")
	assert.EqualError(t, err, "invalid s3 path s3://ag-ge")
}

func TestFileReader(t *testing.T) {
	// Create a temporary file and write some data to it
	file, err := os.CreateTemp("/tmp", "pathioFileReaderTest")
	assert.Nil(t, err)
	text := "fileReaderTest"
	_ = os.WriteFile(file.Name(), []byte(text), 0644)

	reader, err := Reader(file.Name())
	assert.Nil(t, err)
	line, _, err := bufio.NewReader(reader).ReadLine()
	assert.Nil(t, err)
	assert.Equal(t, string(line), text)
}

func TestWriteToFilePath(t *testing.T) {
	file, err := os.CreateTemp("/tmp", "writeToPathTest")
	assert.Nil(t, err)
	defer os.Remove(file.Name())

	assert.Nil(t, Write(file.Name(), []byte("testout")))
	output, err := os.ReadFile(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, "testout", string(output))
}

func TestS3Calls(t *testing.T) {
	testCases := []struct {
		desc     string
		testCase func(svc *Mocks3Handler, t *testing.T)
	}{
		{
			desc: "GetRegionForBucketSuccess",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				name, region := "bucket", "region"
				output := s3.HeadBucketOutput{BucketRegion: &region}
				params := s3.HeadBucketInput{Bucket: aws.String(name)}
				svc.EXPECT().HeadBucket(gomock.Any(), &params).Return(&output, nil)
				foundRegion, _ := getRegionForBucket(context.TODO(), svc, name)
				assert.Equal(t, region, foundRegion)
			},
		},
		{
			desc: "GetRegionForBucketDefault",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				name := "bucket"
				output := s3.HeadBucketOutput{BucketRegion: nil}
				svc.EXPECT().HeadBucket(gomock.Any(), gomock.Any()).Return(&output, nil)
				foundRegion, _ := getRegionForBucket(context.TODO(), svc, name)
				assert.Equal(t, "us-east-1", foundRegion)
			},
		},
		{
			desc: "GetRegionForBucketError",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				name, err := "bucket", "Error!"
				output := s3.HeadBucketOutput{BucketRegion: nil}
				svc.EXPECT().HeadBucket(gomock.Any(), gomock.Any()).Return(&output, errors.New(err))
				_, foundErr := getRegionForBucket(context.TODO(), svc, name)
				assert.Equal(t, foundErr, fmt.Errorf("failed to get location for bucket '%s', %s", name, err))
			},
		},
		{
			desc: "S3FileReaderSuccess",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key, value := "bucket", "key", "value"
				reader := io.NopCloser(bytes.NewBuffer([]byte(value)))
				output := s3.GetObjectOutput{Body: reader}
				params := s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				}
				svc.EXPECT().GetObject(gomock.Any(), &params).Return(&output, nil)
				foundReader, _ := s3FileReader(context.TODO(), s3Connection{svc, bucket, key})
				body := make([]byte, len(value))
				_, err := foundReader.Read(body)
				assert.NoError(t, err)
				assert.Equal(t, string(body), value)
			},
		},
		{
			desc: "S3FileReaderError",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key, err := "bucket", "key", "Error!"
				params := s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				}
				output := s3.GetObjectOutput{}
				svc.EXPECT().GetObject(gomock.Any(), &params).Return(&output, errors.New(err))
				_, foundErr := s3FileReader(context.TODO(), s3Connection{svc, bucket, key})
				assert.Equal(t, foundErr.Error(), err)
			},
		},
		{
			desc: "S3FileWriterSuccess",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key := "bucket", "key"
				input := bytes.NewReader(make([]byte, 0))
				output := s3.PutObjectOutput{}
				params := s3.PutObjectInput{
					Bucket:               aws.String(bucket),
					Key:                  aws.String(key),
					Body:                 input,
					ServerSideEncryption: "AES256",
				}
				svc.EXPECT().PutObject(gomock.Any(), &params).Return(&output, nil)
				foundErr := writeToS3(context.TODO(), s3Connection{svc, bucket, key}, input, false)
				assert.Equal(t, foundErr, nil)
			},
		},
		{
			desc: "S3FileWriterError",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key, err := "bucket", "key", "Error!"
				input := bytes.NewReader(make([]byte, 0))
				output := s3.PutObjectOutput{}
				params := s3.PutObjectInput{
					Bucket:               aws.String(bucket),
					Key:                  aws.String(key),
					Body:                 input,
					ServerSideEncryption: "AES256",
				}
				svc.EXPECT().PutObject(gomock.Any(), &params).Return(&output, errors.New(err))
				foundErr := writeToS3(context.TODO(), s3Connection{svc, bucket, key}, input, false)
				assert.Equal(t, foundErr.Error(), err)
			},
		},
		{
			desc: "S3FileWriterSuccessNoEncryption",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key := "bucket", "key"
				input := bytes.NewReader(make([]byte, 0))
				output := s3.PutObjectOutput{}
				params := s3.PutObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
					Body:   input,
				}
				svc.EXPECT().PutObject(gomock.Any(), &params).Return(&output, nil)
				foundErr := writeToS3(context.TODO(), s3Connection{svc, bucket, key}, input, true)
				assert.Equal(t, foundErr, nil)
			},
		},
		{
			desc: "S3ListFiles",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key := "bucket", "key"
				output := []*s3.ListObjectsV2Output{
					{
						Contents: []s3Types.Object{
							{Key: aws.String("file1")},
						},
						CommonPrefixes: []s3Types.CommonPrefix{
							{Prefix: aws.String("prefix/")},
						},
						IsTruncated: aws.Bool(false),
					},
				}

				params := s3.ListObjectsV2Input{
					Bucket:    aws.String(bucket),
					Prefix:    aws.String(key),
					Delimiter: aws.String("/"),
				}

				svc.EXPECT().ListAllObjects(gomock.Any(), &params).Return(output, nil)
				files, err := lsS3(context.TODO(), s3Connection{svc, bucket, key})
				assert.NoError(t, err)
				assert.Equal(t, []string{"prefix/", "file1"}, files)
			},
		},
		{
			desc: "S3ListFilesRecurse",
			testCase: func(svc *Mocks3Handler, t *testing.T) {
				bucket, key := "bucket", "key"

				output := []*s3.ListObjectsV2Output{
					{
						Contents: []s3Types.Object{
							{Key: aws.String("file1")},
						},
						CommonPrefixes: []s3Types.CommonPrefix{
							{Prefix: aws.String("prefix/")},
							{Prefix: aws.String("prefix2/")},
						},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("file1"),
					},
					{
						Contents: []s3Types.Object{
							{Key: aws.String("file2")},
						},
						CommonPrefixes: []s3Types.CommonPrefix{
							{Prefix: aws.String("prefix2/")},
						},
						IsTruncated: aws.Bool(false),
					},
				}

				params := s3.ListObjectsV2Input{
					Bucket:    aws.String(bucket),
					Prefix:    aws.String(key),
					Delimiter: aws.String("/"),
				}

				svc.EXPECT().ListAllObjects(gomock.Any(), &params).Return(output, nil)

				files, err := lsS3(context.TODO(), s3Connection{svc, bucket, key})
				assert.NoError(t, err)
				assert.Equal(t, []string{"prefix/", "prefix2/", "file1", "file2"}, files)
			},
		},
	}
	for _, spec := range testCases {
		t.Run(spec.desc, func(t *testing.T) {
			c := gomock.NewController(t)
			svc := NewMocks3Handler(c)
			spec.testCase(svc, t)
			c.Finish()
		})
	}
}
