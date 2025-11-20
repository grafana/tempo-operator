# Build the manager binary
FROM golang:1.24@sha256:58364c0a7d53e26b6e5b83cf186b4e9d5fa078c4f5e728f9d9a7b972f689a84a as builder

WORKDIR /workspace
# Cache tool dependencies
COPY Makefile Makefile
RUN make controller-gen kustomize
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .


# Build
ARG OPERATOR_VERSION
ARG TEMPO_VERSION
RUN make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:e8a4044e0b4ae4257efa45fc026c0bc30ad320d43bd4c1a7d5271bd241e386d0
WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
