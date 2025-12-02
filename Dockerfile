
ARG distroless_tag=latest

FROM golang:1.25.4-trixie AS builder

RUN apt-get update && apt-get install -y build-essential && apt clean
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./app/ ./app/
COPY ./cmd/ ./cmd/
RUN ls -l
RUN mkdir -p bin
RUN go build -tags json1,fts5 -o bin/dhee ./cmd/dhee
COPY ./data/ ./data/
RUN bin/dhee preprocess --input ./data --output ./data/ --embeddings-file data/rv.emb.jsonl
RUN bin/dhee index --data-dir ./data --store sqlite

FROM gcr.io/distroless/base-debian12:${distroless_tag}
COPY --from=builder /usr/lib/x86_64-linux-gnu/libsqlite3.so.* /usr/lib/x86_64-linux-gnu/
WORKDIR /app
COPY --from=builder /app/bin/dhee /app/bin/dhee
COPY --from=builder /app/data/dhee.db /app/data/dhee.db
COPY ./data/config.json /app/data/config.json

ENTRYPOINT ["./bin/dhee", "server", "--data-dir", "./data", "--store", "sqlite", "--address", "0.0.0.0"]

EXPOSE 8080
