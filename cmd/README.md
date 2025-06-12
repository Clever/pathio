# p3
--

The p3 executable exist to make it easier to manually test pathio.

## Usage

From the project root:

```
make build
./build/p3 upload s3://BUCKET/KEY /L)CAL_FILE
./build/p3 download s3://BUCKET/KEY /LOCAL_FILE
```

Notes for testing:

* Order matters for file names
* The optional flag `--profile=` can be used for allowing p3 to authenticate using a profile instead of environment
  variables.
    * This does require you to have profile
      under [~/.aws/credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html#cli-configure-files-where)
