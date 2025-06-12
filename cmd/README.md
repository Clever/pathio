# p3
--

The p3 executable exist to make it easier to manually test pathio.

## Usage

From the project root:

```
make build

# Uploading a local file to s3
./build/p3 upload s3://BUCKET/KEY /LOCAL_FILE

# Downloading an s3 file
./build/p3 download s3://BUCKET/KEY /LOCAL_FILE

# Listing contents of an s3 bucket or local file path
./build/p3 list s3://BUCKET/KEY
./build/p3 list LOCAL_FILE

# Check if the s3 or local file path exists
./build/p3 exists s3://BUCKET/KEY
./build/p3 exists LOCAL_FILE

# Delete the s3 object or local file
./build/p3 delete s3://BUCKET/KEY
./build/p3 delete LOCAL_FILE

# Write the contents of the provided string to an s3 object or local file
./build/p3 write "hello world" s3://BUCKET/KEY
./build/p3 write "hello world" LOCAL_FILE

```

Notes for testing:

* The optional flag `--profile=` can be used for allowing p3 to authenticate using a profile instead of environment
  variables.
    * This does require you to have profile
      under [~/.aws/credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html#cli-configure-files-where)

### CLI --help

```
usage: p3 [<flags>] <command> [<args> ...]

Flags:
  --[no-]help   Show context-sensitive help (also try --help-long and --help-man).
  --profile=""  AWS profile to use in lieu of the AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID environment variables

Commands:
help [<command>...]
    Show help.

list <file_path>
    list contents of an S3 path

download <s3_path> <local_path>
    download contents of an S3 path to a local file

upload <s3_path> <local_path>
    upload contents of a local file to an S3 path

delete <file_path>
    delete contents of an S3 path

exists <path>
    check if the s3 path exists

write <contents> <destination_path>
    copy contents of a string to a file

```