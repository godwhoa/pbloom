FROM rust:1.83-bookworm AS builder

# Requirements for pgrx
RUN apt-get update && apt-get install -y \
    build-essential \
    libreadline-dev \
    zlib1g-dev \
    flex \
    bison \
    libxml2-dev \
    libxslt-dev \
    libssl-dev \
    libxml2-utils \
    xsltproc \
    ccache \
    pkg-config

# Install PostgreSQL 16 via Postgres repo
RUN apt-get update && apt-get install -y wget ca-certificates gnupg \
    && wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql-archive-keyring.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/postgresql-archive-keyring.gpg] http://apt.postgresql.org/pub/repos/apt/ bookworm-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y postgresql-16 postgresql-server-dev-16


RUN cargo install --locked cargo-pgrx

RUN mkdir /pbloom /pbloom-pg16

WORKDIR /pbloom

COPY . .

RUN cd pg && cargo pgrx init --pg16 `which pg_config` && cargo pgrx package \
    && cp /pbloom/pg/target/release/pbloompg-pg16/usr/lib/postgresql/16/lib/pbloompg* /pbloom-pg16 \
    && cp /pbloom/pg/target/release/pbloompg-pg16/usr/share/postgresql/16/extension/pbloompg* /pbloom-pg16

FROM ghcr.io/cloudnative-pg/postgresql:16.0

COPY --from=builder /pbloom-pg16/*.so /usr/lib/postgresql/16/lib
COPY --from=builder /pbloom-pg16/*.control /usr/share/postgresql/16/extension