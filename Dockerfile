# Build the manager binary
FROM golang:1.26@sha256:6e59e17a20f13c102e49d82dffab66d41cffe6c266b225d82794328aea859ab5 AS builder

WORKDIR /workspace
# Cache tool dependencies
COPY Makefile Makefile
RUN make controller-gen kustomize

# Copy the go source
COPY . .

# Build
ARG OPERATOR_VERSION
ARG TEMPO_VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5
WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
