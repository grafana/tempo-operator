# Build the manager binary
# Digest pinned to golang:1.25 as of 2026-05-26; Dependabot will raise PRs when it changes.
FROM golang:1.25@sha256:c138bff780910acf4254ab3a6f7ff0f64bbd841f27bd82bfa986fe122c109538 AS builder

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


FROM registry.access.redhat.com/ubi9-minimal:9.8

RUN INSTALL_PKGS=" \
  rsync \
  tar \
  " && \
  microdnf install -y $INSTALL_PKGS && \
  microdnf clean all

COPY --from=builder /workspace/bin/must-gather /usr/bin/must-gather

USER 65532:65532

ENTRYPOINT ["/usr/bin/must-gather"]
