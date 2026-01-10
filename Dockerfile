
# can use nonroot-debug for debugging
ARG distroless_tag=nonroot

FROM golang:1.25.4-trixie AS builder
ARG build_tags="json1,fts5,native_sqlite"
RUN apt-get update && apt-get install -y build-essential && apt-get clean
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./app/ ./app/
COPY ./cmd/ ./cmd/
RUN ls -l
RUN mkdir -p bin
RUN go build -tags "${build_tags}" -o bin/dhee ./cmd/dhee
COPY ./data/ ./data/
RUN bin/dhee preprocess --input ./data --output ./data/ --embeddings-file data/rv.emb.jsonl
RUN bin/dhee index --data-dir ./data --store sqlite

FROM gcr.io/distroless/base-debian12:${distroless_tag}

COPY --from=builder /usr/lib/x86_64-linux-gnu/libsqlite3.so.* /usr/lib/x86_64-linux-gnu/
USER nonroot:nonroot
WORKDIR /app

COPY --chown=nonroot:nonroot --from=builder /app/bin/dhee /app/bin/dhee
COPY --chown=nonroot:nonroot --from=builder /app/data/dhee.db /app/data/dhee.db
COPY --chown=nonroot:nonroot ./data/config.json /app/data/config.json

CMD ["./bin/dhee", "server", "--data-dir", "./data", "--store", "sqlite", "--address", "0.0.0.0", "--cert-dir", "/app/certs", "--acme", "--rate-limit", "8", "--global-rate-limit", "64"]

EXPOSE 8080
