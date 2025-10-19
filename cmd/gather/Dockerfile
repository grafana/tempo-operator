# Build the manager binary
FROM golang:1.24 as builder

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

RUN make must-gather


FROM registry.access.redhat.com/ubi9-minimal:9.2

RUN INSTALL_PKGS=" \
  rsync \
  tar \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all

COPY --from=builder /workspace/bin/must-gather /usr/bin/must-gather

USER 65532:65532

ENTRYPOINT ["/usr/bin/must-gather"]
