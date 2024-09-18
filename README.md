# S3 File Remover

The S3 File Remover is a command-line utility designed to bulk delete objects from Amazon S3 or S3-compatible storage systems (like MinIO) based on their last modified date. This tool is particularly useful for managing large numbers of objects and performing cleanup operations on your storage buckets.

See [s3-file-generator](https://github.com/zinrai/s3-file-generator) for file generation to s3.

## Features

- Bulk deletion of objects older than a specified date
- Support for both Amazon S3 and S3-compatible storage systems (e.g., MinIO)
- Concurrent processing with customizable number of workers for improved performance
- Flexible date input format, including RFC3339 and other common formats
- Configurable maximum number of keys to delete in a single operation
- Detailed logging of deletion progress and results

## Installation

Build the tool:

```
$ go build
```

## Usage

The basic syntax for using the tool is:

```
$ s3-file-remover -bucket <bucket-name> -date <date> [options]
```

### Examples

1. Delete objects from an AWS S3 bucket older than January 1, 2023:
   ```
   $ s3-file-remover -bucket my-bucket -date 2023-01-01 -region us-west-2
   ```

2. Use with MinIO, specifying endpoint and credentials:
   ```
   $ s3-file-remover -bucket test-bucket -date 2023-01-01 -workers 20 -endpoint http://localhost:9000 -region us-east-1 -access-key mykey -secret-key mysecret
   ```

3. Use a specific RFC3339 formatted date and custom max keys per delete operation:
   ```
   $ s3-file-remover -bucket my-bucket -date "2023-01-01T00:00:00Z" -max-keys 500
   ```

## Output

Upon completion, the tool will output a summary line similar to:

```
2024/09/18 12:11:19 Operation complete. Deleted 30300/30300 objects in 2m15s
```

This line indicates the total number of objects deleted, the total number of objects that were targeted for deletion, and the total time taken for the operation.

## Notes

- When using with AWS S3 (no `-endpoint` specified):
  - The tool uses the AWS SDK's default credential chain (environment variables, AWS credentials file, IAM role).
  - Ensure you have the necessary permissions to list and delete objects in the specified bucket.
- When using with S3-compatible storage (`-endpoint` specified):
  - You must provide the `-access-key` and `-secret-key`.
  - The tool uses path-style addressing.
- The deletion process is irreversible. Use this tool with caution and verify the date parameter before running.
- For large buckets, consider adjusting the number of workers and max-keys to optimize performance.
- The tool now supports both AWS S3 and S3-compatible services, automatically adjusting its behavior based on whether an endpoint is specified.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
