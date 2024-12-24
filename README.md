# pbloom

> A project focused on creating a portable bloom filter library designed for use across various programming languages. Specifically, the initial goal was to develop the capability to create and serialize bloom filters using Go, and subsequently utilize them within a Postgres extension implemented in Rust.


# Current State

- Go and Rust libraries for creating, populating, querying, and serializing bloom filters.
- Rust-based Postgres extension utilizing the bloom filter library.
- Docker image with PostgreSQL 16.0 and pbloompg extension.
- Example containing: create and serialize in Go, insert into PG, query with pbloompg.

# Future

- Extend support to additional programming languages.
- Incorporate other data structures like xor, ribbon, and ngram bloom filters.
- Performance testing and optimization.
- Proper specifications.
