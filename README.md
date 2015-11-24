# pathio
--
    import "github.com/Clever/pathio"

Package pathio is a package that allows writing to and reading from different
types of paths transparently. It supports two types of paths:

    1. Local file paths
    2. S3 File Paths (s3://bucket/key)

Note that using s3 paths requires setting two environment variables

    1. AWS_SECRET_ACCESS_KEY
    2. AWS_ACCESS_KEY_ID

## Usage

#### func  Reader

```go
func Reader(path string) (rc io.ReadCloser, err error)
```
Reader Calls DefaultClient's Reader method.

#### func  Write

```go
func Write(path string, input []byte) error
```
Write Calls DefaultClient's Write method.

#### func  WriteReader

```go
func WriteReader(path string, input io.ReadSeeker) error
```
WriteReader Calls DefaultClient's WriteReader method.

#### type AWSClient

```go
type AWSClient struct {
}
```

AWSClient is the pathio client used to access the local file system and S3. It
includes an option to disable S3 encryption. To disable S3 encryption, create a
new Client and call it directly: `&AWSClient{disableS3Encryption:
true}.Write(...)`

#### func (*AWSClient) Reader

```go
func (c *AWSClient) Reader(path string) (rc io.ReadCloser, err error)
```
Reader returns an io.Reader for the specified path. The path can either be a
local file path or an S3 path. It is the caller's responsibility to close rc.

#### func (*AWSClient) Write

```go
func (c *AWSClient) Write(path string, input []byte) error
```
Write writes a byte array to the specified path. The path can be either a local
file path or an S3 path.

#### func (*AWSClient) WriteReader

```go
func (c *AWSClient) WriteReader(path string, input io.ReadSeeker) error
```
WriteReader writes all the data read from the specified io.Reader to the output
path. The path can either a local file path or an S3 path.

#### type Client

```go
type Client interface {
	Write(path string, input []byte) error
	Reader(path string) (rc io.ReadCloser, err error)
	WriteReader(path string, input io.ReadSeeker) error
}
```

Client is the interface exposed by pathio to both S3 and the filesystem.

```go
var DefaultClient Client = &AWSClient{}
```
DefaultClient is the default pathio client called by the Reader, Writer, and
WriteReader methods. It has S3 encryption enabled.

#### type MockClient

```go
type MockClient struct {
	// Filesystem holds a theoretical S3 bucket for mocking puproses
	Filesystem map[string]string
	// These are errors that will be returned by the mocked methods if set
	WriteErr, ReaderErr, WriteReaderErr error
}
```

MockClient mocks out an S3 bucket

#### func (*MockClient) Reader

```go
func (m *MockClient) Reader(path string) (rc io.ReadCloser, err error)
```
Reader returns MockClient.ReaderErr if set, otherwise reads from internal data.

#### func (*MockClient) Write

```go
func (m *MockClient) Write(path string, input []byte) error
```
Write returns MockClient.WriteErr if set, otherwise stores the data internally.

#### func (*MockClient) WriteReader

```go
func (m *MockClient) WriteReader(path string, input io.ReadSeeker) error
```
WriteReader returns MockClient.WriteReaderErr if set, otherwise stores the data
internally.
