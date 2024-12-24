# pbloom

> A project focused on creating a portable bloom filter library designed for use across various programming languages. Specifically, the initial goal was to develop the capability to create and serialize bloom filters using Go, and subsequently utilize them within a Postgres extension implemented in Rust.

# Current State

- We have libraries for Go and Rust that can create, populate, query and serialize bloom filters.
- We have a Postgres extension in Rust, which utilizes the Rust library to create, populate, and query bloom filters.
- We have docker image for PG based on `cloudnative-pg/postgresql:16.0` which includes the pbloompg extension.
- We have an example where we create/populate/serialize in Go, insert it into PG, and query it using the pbloompg extension.

# Future

- Add support for more languages.
- Add support for more similar data structures, like xor filters, ribbon filters, ngram bloom filters, etc.
- Performance testing and optimization.
- Proper specification