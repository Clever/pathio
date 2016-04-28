# pathio

[![GoDoc](https://godoc.org/gopkg.in/Clever/pathio.v3?status.svg)](https://godoc.org/gopkg.in/Clever/pathio.v3)

--
    import "gopkg.in/Clever/pathio.v3"

Package pathio is a package that allows writing to and reading from different
types of paths transparently. It supports two types of paths:

    1. Local file paths
    2. S3 File Paths (s3://bucket/key)

Note that using s3 paths requires setting two environment variables

    1. AWS_SECRET_ACCESS_KEY
    2. AWS_ACCESS_KEY_ID

