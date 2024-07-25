FROM golang:1.22-bookworm AS builder
ARG GOGC=off

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -o /manager './cmd/manager'

FROM gcr.io/distroless/base-debian12:nonroot
USER nonroot:nonroot
WORKDIR /
COPY --from=builder /manager /manager
CMD ["/manager"]
