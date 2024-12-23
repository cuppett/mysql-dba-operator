# Build the manager binary
FROM ghcr.io/cuppett/golang:1.23 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY orm/ orm/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM ghcr.io/cuppett/fedora-minimal:latest

LABEL maintainer="Stephen Cuppett <steve@cuppett.com>" \
      org.opencontainers.image.title="mysql-dba-operator" \
      org.opencontainers.image.description="Operator for managing MySQL connections, databases & users" \
      org.opencontainers.image.source="https://github.com/cuppett/mysql-dba-operator"

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
