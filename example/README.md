# Go & PG Bloom Filter Example

> In this example, we create/populate bloom filters then serialize them and insert into PG. We then use the pbloompg extension to query the bloom filters.

## Prerequisites

*   Go
*   Docker & Docker Compose

## Run

```bash
docker-compose up -d
go run main.go
Files with rows name=John: [file1.parquet]
Files with rows date=2020-01-01: [file1.parquet file2.parquet]
```