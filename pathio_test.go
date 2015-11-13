package pathio

import (
	"bufio"
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

type mockedS3Api struct {
	mock.Mock
}

func (m *mockedS3Api) GetBucketLocation(input *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.GetBucketLocationOutput), args.Error(1)
}

func TestGetRegionForBucketSuccess(t *testing.T) {
	svc := mockedS3Api{}
	name, region := "bucket", "region"
	output := s3.GetBucketLocationOutput{LocationConstraint: aws.String(region)}
	params := s3.GetBucketLocationInput{Bucket: aws.String(name)}
	svc.On("GetBucketLocation", params).Return(&output, nil)
	foundRegion, _ := getRegionForBucket(&svc, name)
	assert.Equal(t, region, foundRegion)
	svc.AssertExpectations(t)
}

func TestGetRegionForBucketError(t *testing.T) {
	svc := mockedS3Api{}
	name, err := "bucket", "Error!"
	output := s3.GetBucketLocationOutput{LocationConstraint: nil}
	svc.On("GetBucketLocation", mock.Anything).Return(&output, errors.New(err))
	_, foundErr := getRegionForBucket(&svc, name)
	assert.Equal(t, foundErr, fmt.Errorf("Failed to get location for bucket '%s', %s", name, err))
	svc.AssertExpectations(t)
}
