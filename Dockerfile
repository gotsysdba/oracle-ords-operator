# Build the manager binary
FROM container-registry.oracle.com/os/oraclelinux:9-slim AS builder
# Match what is in go.mod (restricted to latest base image version)
ARG GO_VERSION=1.21.9
RUN \
    microdnf install go-toolset-${GO_VERSION}; \
    microdnf update
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Runtime
FROM container-registry.oracle.com/os/oraclelinux:9-slim
WORKDIR /
COPY --from=builder /workspace/manager .
COPY internal/controller/ords_init.sh .
RUN useradd -u 10001 nonroot
USER 10001:10001

ENTRYPOINT ["/manager"]