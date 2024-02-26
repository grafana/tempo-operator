# The Dockerfile's resulting image is purpose-built for executing Tempo Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI. 

FROM golang:1.20

# Copy the repository files
COPY . /tmp/tempo-operator

WORKDIR /tmp

# Install dependencies required by test cases and debugging
RUN apt-get update && apt-get install -y jq vim libreadline-dev

# Install chainsaw
RUN go install github.com/kyverno/chainsaw@v0.1.4

# Install kubectl and oc
RUN curl -L -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xvzf oc.tar.gz \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/

# Set the working directory
WORKDIR /tmp/tempo-operator

# Build required binaries
RUN make build
