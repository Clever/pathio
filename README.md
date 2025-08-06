# pathio

[![GoDoc](https://godoc.org/github.com/Clever/pathio/v5?status.svg)](https://godoc.org/github.com/Clever/pathio/v5)

```
go get "github.com/Clever/pathio/v5"
```

Package pathio is a package that allows writing to and reading from different
types of paths transparently. It supports two types of paths:

    1. Local file paths
    2. S3 File Paths (s3://bucket/key)

Note that using s3 paths requires setting two environment variables

    1. AWS_SECRET_ACCESS_KEY
    2. AWS_ACCESS_KEY_ID

## Usage

Pathio has a very easy to use interface, with 5 main functions:

```
import "github.com/Clever/pathio/v5"
var err error
```

### Initialization

Initializing a new Client:

```
    ctx := context.Background()
	// Load the default AWS configuration
	awsConfig, err := awsV2Config.LoadDefaultConfig(ctx, awsV2Config.WithDefaultRegion("us-east-1"))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	pathioClient := pathio.NewClient(ctx, &awsConfig)

```

Using the Default Client (Import the Package): 

```
    import (
        "github.com/Clever/pathio/v5"
    )
```

```
    pathioClient := pathio.DefaultClient
    arcReader, err := pathioClient.Reader(wd.Input.Archive)
```

### ListFiles

```
// func ListFiles(path string) ([]string, error)
files, err = pathio.ListFiles("s3://bucket/my/key") // s3
files, err = pathio.ListFiles("/home/me")           // local
```

### Write / WriteReader

```
// func Write(path string, input []byte) error
toWrite := []byte("hello world\n")
err = pathio.Write("s3://bucket/my/key", toWrite)   // s3
err = pathio.Write("/home/me/hello_world", toWrite) // local

// func WriteReader(path string, input io.ReadSeeker) error
toWriteReader, err := os.Open("test.txt") // this implements Read and Seek
err = pathio.WriteReader("s3://bucket/my/key", toWriteReader)   // s3
err = pathio.WriteReader("/home/me/hello_world", toWriteReader) // local
```

### Read

```
// func Reader(path string) (rc io.ReadCloser, err error)
reader, err = pathio.Reader("s3://bucket/key/to/read") // s3
reader, err = pathio.Reader("/home/me/file/to/read")   // local
```

### Delete

```
// func Delete(path string) error
err = pathio.Delete("s3://bucket/key/to/read") // s3
err = pathio.Delete("/home/me/file/to/read")   // local
```
