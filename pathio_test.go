package pathio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	assert.EqualError(t, err, "Invalid s3 path s3://")

	_, _, err = parseS3Path("s3://ag-ge")
	assert.EqualError(t, err, "Invalid s3 path s3://ag-ge")
}

func TestFileReader(t *testing.T) {
	// Create a temporary file and write some data to it
	file, err := ioutil.TempFile("/tmp", "pathioFileReaderTest")
	assert.Nil(t, err)
	text := "fileReaderTest"
	ioutil.WriteFile(file.Name(), []byte(text), 0644)

	reader, err := Reader(file.Name())
	assert.Nil(t, err)
	line, _, err := bufio.NewReader(reader).ReadLine()
	assert.Nil(t, err)
	assert.Equal(t, string(line), text)
}

func TestWriteToFilePath(t *testing.T) {
	file, err := ioutil.TempFile("/tmp", "writeToPathTest")
	assert.Nil(t, err)
	defer os.Remove(file.Name())

	assert.Nil(t, Write(file.Name(), []byte("testout")))
	output, err := ioutil.ReadFile(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, "testout", string(output))
}

type mockedS3Client struct {
	mock.Mock
}

func (m *mockedS3Client) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.GetBucketLocationOutput), args.Error(1)
}

func (m *mockedS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mockedS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func TestGetRegionForBucketSuccess(t *testing.T) {
	svc := mockedS3Client{}
	name, region := "bucket", "region"
	output := s3.GetBucketLocationOutput{LocationConstraint: aws.String(region)}
	params := s3.GetBucketLocationInput{Bucket: aws.String(name)}
	svc.On("GetBucketLocation", &params).Return(&output, nil)
	foundRegion, _ := getRegionForBucket(&svc, name)
	assert.Equal(t, region, foundRegion)
	svc.AssertExpectations(t)
}

func TestGetRegionForBucketDefault(t *testing.T) {
	svc := mockedS3Client{}
	name := "bucket"
	output := s3.GetBucketLocationOutput{LocationConstraint: nil}
	svc.On("GetBucketLocation", mock.Anything).Return(&output, nil)
	foundRegion, _ := getRegionForBucket(&svc, name)
	assert.Equal(t, "us-east-1", foundRegion)
	svc.AssertExpectations(t)
}

func TestGetRegionForBucketError(t *testing.T) {
	svc := mockedS3Client{}
	name, err := "bucket", "Error!"
	output := s3.GetBucketLocationOutput{LocationConstraint: nil}
	svc.On("GetBucketLocation", mock.Anything).Return(&output, errors.New(err))
	_, foundErr := getRegionForBucket(&svc, name)
	assert.Equal(t, foundErr, fmt.Errorf("Failed to get location for bucket '%s', %s", name, err))
	svc.AssertExpectations(t)
}

func TestS3FileReaderSuccess(t *testing.T) {
	svc := mockedS3Client{}
	bucket, key, value := "bucket", "key", "value"
	reader := ioutil.NopCloser(bytes.NewBuffer([]byte(value)))
	output := s3.GetObjectOutput{Body: reader}
	params := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	svc.On("GetObject", &params).Return(&output, nil)
	foundReader, _ := s3FileReader(s3Connection{&svc, bucket, key})
	body := make([]byte, len(value))
	foundReader.Read(body)
	assert.Equal(t, string(body), value)
	svc.AssertExpectations(t)
}

func TestS3FileReaderError(t *testing.T) {
	svc := mockedS3Client{}
	bucket, key, err := "bucket", "key", "Error!"
	params := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	output := s3.GetObjectOutput{}
	svc.On("GetObject", &params).Return(&output, errors.New(err))
	_, foundErr := s3FileReader(s3Connection{&svc, bucket, key})
	assert.Equal(t, foundErr.Error(), err)
	svc.AssertExpectations(t)
}

func TestS3FileWriterSuccess(t *testing.T) {
	svc := mockedS3Client{}
	bucket, key := "bucket", "key"
	input := bytes.NewReader(make([]byte, 0))
	output := s3.PutObjectOutput{}
	params := s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(key),
		Body:                 input,
		ServerSideEncryption: aws.String("AES256"),
	}
	svc.On("PutObject", &params).Return(&output, nil)
	foundErr := writeToS3(s3Connection{&svc, bucket, key}, input, false)
	assert.Equal(t, foundErr, nil)
	svc.AssertExpectations(t)
}

func TestS3FileWriterError(t *testing.T) {
	svc := mockedS3Client{}
	bucket, key, err := "bucket", "key", "Error!"
	input := bytes.NewReader(make([]byte, 0))
	output := s3.PutObjectOutput{}
	params := s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(key),
		Body:                 input,
		ServerSideEncryption: aws.String("AES256"),
	}
	svc.On("PutObject", &params).Return(&output, errors.New(err))
	foundErr := writeToS3(s3Connection{&svc, bucket, key}, input, false)
	assert.Equal(t, foundErr.Error(), err)
	svc.AssertExpectations(t)
}

func TestS3FileWriterSuccessNoEncryption(t *testing.T) {
	svc := mockedS3Client{}
	bucket, key := "bucket", "key"
	input := bytes.NewReader(make([]byte, 0))
	output := s3.PutObjectOutput{}
	params := s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   input,
	}
	svc.On("PutObject", &params).Return(&output, nil)
	foundErr := writeToS3(s3Connection{&svc, bucket, key}, input, true)
	assert.Equal(t, foundErr, nil)
	svc.AssertExpectations(t)
}
