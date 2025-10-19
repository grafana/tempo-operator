# The Dockerfile's resulting image is purpose-built for executing Tempo Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI. 

FROM golang:1.24

# Copy the repository files
COPY . /tmp/tempo-operator

WORKDIR /tmp

# Install dependencies required by test cases and debugging
RUN apt-get update && apt-get install -y jq vim libreadline-dev unzip

# Install kuttl
RUN curl -LO https://github.com/kudobuilder/kuttl/releases/download/v0.15.0/kubectl-kuttl_0.15.0_linux_x86_64 \
    && chmod +x kubectl-kuttl_0.15.0_linux_x86_64 \
    && mv kubectl-kuttl_0.15.0_linux_x86_64 /usr/local/bin/kuttl

# Install chainsaw
RUN curl -L -o chainsaw.tar.gz https://github.com/kyverno/chainsaw/releases/download/v0.2.12/chainsaw_linux_amd64.tar.gz \
    && tar -xvzf chainsaw.tar.gz \
    && chmod +x chainsaw \
    && mv chainsaw /usr/local/bin/

# Install kubectl and oc
RUN curl -L -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xvzf oc.tar.gz \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/

# Install AWS CLI
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip \
    && ./aws/install

# Install Azure CLI
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# Install Google Cloud CLI
RUN curl -LO https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz \
    && tar -xf google-cloud-cli-linux-x86_64.tar.gz \
    && ./google-cloud-sdk/install.sh -q

# Add gcloud to PATH
ENV PATH="/tmp/google-cloud-sdk/bin:${PATH}"

# Set the working directory
WORKDIR /tmp/tempo-operator

# Build required binaries
RUN make build
