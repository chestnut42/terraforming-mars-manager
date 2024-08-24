FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Build most root packages first and cache in a layer
COPY ./internal/auth ./internal/auth
RUN go build ./internal/auth/...

COPY ./internal/framework ./internal/framework
COPY ./internal/database ./internal/database
COPY ./internal/storage ./internal/storage
RUN go build ./internal/storage/...

COPY . ./
RUN go build -o /manager './cmd/manager'

FROM gcr.io/distroless/base-debian12:nonroot
USER nonroot:nonroot
WORKDIR /
COPY --from=builder /manager /manager
CMD ["/manager"]
