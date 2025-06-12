package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	awsV2Config "github.com/aws/aws-sdk-go-v2/config"

	"github.com/Clever/pathio/v5"
)

var (
	awsProfile = kingpin.Flag("profile", "AWS profile to use in lieu of the AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID environment variables").Default("").String()

	listCommand = kingpin.Command("list", "list contents of an S3 path")
	listPath    = listCommand.Arg("file_path", "S3 or local path to list the contents").Required().String()

	downloadCommand   = kingpin.Command("download", "download contents of an S3 path to a local file")
	downloadS3Path    = downloadCommand.Arg("s3_path", "S3 path to download").Required().String()
	downloadLocalPath = downloadCommand.Arg("local_path", "local file to write to").Required().String()

	uploadCommand   = kingpin.Command("upload", "upload contents of a local file to an S3 path")
	uploadS3Path    = uploadCommand.Arg("s3_path", "S3 path to upload").Required().String()
	uploadLocalPath = uploadCommand.Arg("local_path", "local file to write to").Required().String()

	deleteCommand = kingpin.Command("delete", "delete contents of an S3 path")
	deletePath    = deleteCommand.Arg("file_path", "S3 path or local file path to delete").Required().String()

	existsCommand = kingpin.Command("exists", "check if the s3 path exists")
	existsPath    = existsCommand.Arg("path", "S3 path or local file path to check existence of").Required().String()

	writeCommand = kingpin.Command("write", "copy contents of a string to a file")
	contents     = writeCommand.Arg("contents", "string to write to a file").Required().String()
	toPath       = writeCommand.Arg("destination_path", "the local file path or S3 path to be written to").Required().String()
)

// WithSharedProfileConfig is a small wrapper to make the aws profile flag optional. If the flag is used, the s3 client will use this shared profile to authenticate to aws
func WithSharedProfileConfig(profile *string) awsV2Config.LoadOptionsFunc {
	if profile == nil || *profile == "" {
		return nil
	}
	return awsV2Config.WithSharedConfigProfile(*profile)
}

func makePathioS3Client() *pathio.Client {
	ctx := context.Background()
	cfg, err := awsV2Config.LoadDefaultConfig(ctx, WithSharedProfileConfig(awsProfile))
	if err != nil {
		log.Fatalf("error building p3 aws config: %v", err)
	}
	return pathio.NewClient(ctx, &cfg)
}

func main() {
	command := kingpin.Parse()

	switch command {
	// Pathio's ListFiles
	case listCommand.FullCommand():
		listCommandFn()
	// Pathio's Reader
	case downloadCommand.FullCommand():
		downloadCommandFn()
	// Pathio's WriteReader
	case uploadCommand.FullCommand():
		uploadCommandFn()
	// Pathio's Delete
	case deleteCommand.FullCommand():
		deleteCommandFn()
	// Pathio's Exists
	case existsCommand.FullCommand():
		existsCommandFn()
	// Pathio's Write
	case writeCommand.FullCommand():
		writeCommandFn()
	default:
		log.Fatalf("unknown command: %s", command)
	}
}

func listCommandFn() {
	var client pathio.Pathio
	if strings.HasPrefix(*listPath, "s3://") {
		client = makePathioS3Client()
	} else {
		client = pathio.DefaultClient
	}

	results, err := client.ListFiles(*listPath)
	if err != nil {
		log.Fatalf("error list file path: %s", err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func downloadCommandFn() {
	client := makePathioS3Client()

	file, err := os.Create(*downloadLocalPath)
	if err != nil {
		log.Fatalf("Error creating local file: %s", err)
	}
	defer file.Close()
	reader, err := client.Reader(*downloadS3Path)
	if err != nil {
		log.Fatalf("Failed to find s3 file: %s", err)
	}
	defer reader.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		log.Fatalf("Failed to download and write s3 file: %s", err)
	}
	fmt.Printf("Downloaded %s to %s\n", *downloadS3Path, *downloadLocalPath)
}

func uploadCommandFn() {
	client := makePathioS3Client()

	file, err := os.Open(*uploadLocalPath)
	if err != nil {
		log.Fatalf("Error opening file to upload: %s", err)
	}
	defer file.Close()
	err = client.WriteReader(*uploadS3Path, file)
	if err != nil {
		log.Fatalf("Error uploading file: %s", err)
	}
	fmt.Printf("Uploaded %s to %s\n", *uploadLocalPath, *uploadS3Path)
}

func deleteCommandFn() {
	var client pathio.Pathio
	if strings.HasPrefix(*deletePath, "s3://") {
		client = makePathioS3Client()
	} else {
		client = pathio.DefaultClient
	}

	err := client.Delete(*deletePath)
	if err != nil {
		log.Fatalf("error deleting file: %s", err)
	}
	fmt.Printf("Deleted %s successfully\n", *deletePath)
}

func existsCommandFn() {
	var client pathio.Pathio
	if strings.HasPrefix(*existsPath, "s3://") {
		client = makePathioS3Client()
	} else {
		client = pathio.DefaultClient
	}

	exists, err := client.Exists(*existsPath)
	if err != nil {
		log.Fatalf("error checking if file exists: %s", err)
	}
	if exists {
		fmt.Printf("%s exists\n", *existsPath)
	} else {
		fmt.Printf("%s does not exist\n", *existsPath)
	}
}

func writeCommandFn() {
	var client pathio.Pathio
	if strings.HasPrefix(*toPath, "s3://") {
		client = makePathioS3Client()
	} else {
		client = pathio.DefaultClient
	}

	err := client.Write(*toPath, []byte(*contents))
	if err != nil {
		log.Fatalf("error checking if file exists: %s", err)
	}
	fmt.Printf("Wrote contents to: %s\n", *toPath)
}
