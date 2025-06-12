package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

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

func main() {
	command := kingpin.Parse()

	ctx := context.Background()
	cfg, err := awsV2Config.LoadDefaultConfig(ctx, WithSharedProfileConfig(awsProfile))
	if err != nil {
		log.Fatalf("error building p3 aws config: %v", err)
	}
	client := pathio.NewClient(ctx, &cfg)

	switch command {
	// Pathio's ListFiles
	case listCommand.FullCommand():
		listCommandFn(client)
	// Pathio's Reader
	case downloadCommand.FullCommand():
		downloadCommandFn(client)
	// Pathio's WriteReader
	case uploadCommand.FullCommand():
		uploadCommandFn(client)
	// Pathio's Delete
	case deleteCommand.FullCommand():
		deleteCommandFn(client)
	// Pathio's Exists
	case existsCommand.FullCommand():
		existsCommandFn(client)
	// Pathio's Write
	case writeCommand.FullCommand():
		writeCommandFn(client)
	default:
		log.Fatalf("unknown command: %s", command)
	}
}

func listCommandFn(client *pathio.Client) {
	results, err := client.ListFiles(*listPath)
	if err != nil {
		log.Fatalf("error list file path: %s", err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func downloadCommandFn(client *pathio.Client) {
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

func uploadCommandFn(client *pathio.Client) {
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

func deleteCommandFn(client *pathio.Client) {
	err := client.Delete(*deletePath)
	if err != nil {
		log.Fatalf("error deleting file: %s", err)
	}
	fmt.Printf("Deleted %s successfully\n", *deletePath)
}

func existsCommandFn(client *pathio.Client) {
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

func writeCommandFn(client *pathio.Client) {
	err := client.Write(*toPath, []byte(*contents))
	if err != nil {
		log.Fatalf("error checking if file exists: %s", err)
	}
	fmt.Printf("Wrote contents to: %s\n", *toPath)
}
