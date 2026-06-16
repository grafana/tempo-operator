# The Dockerfile's resulting image is purpose-built for executing Tempo Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI.

# Digest pinned to golang:1.25 as of 2026-05-26; Dependabot will raise PRs when it changes.
FROM golang:1.26@sha256:68cb6d68bed024785b69195b89af7ac7a444f27791435f98647edff595aa0479

# Copy the repository files
COPY . /tmp/tempo-operator

WORKDIR /tmp

# Install dependencies required by test cases and debugging.
# gnupg is required for importing the Azure CLI GPG key below.
RUN apt-get update && apt-get install -y jq vim libreadline-dev unzip python3 gnupg

# Install kuttl
# SHA256 from https://github.com/kudobuilder/kuttl/releases/download/v0.15.0/checksums.txt
RUN curl -fL -o kubectl-kuttl \
      https://github.com/kudobuilder/kuttl/releases/download/v0.15.0/kubectl-kuttl_0.15.0_linux_x86_64 \
    && echo "f6edcf22e238fc71b5aa389ade37a9efce596017c90f6994141c45215ba0f862  kubectl-kuttl" | sha256sum -c - \
    && chmod +x kubectl-kuttl \
    && mv kubectl-kuttl /usr/local/bin/kuttl

# Install chainsaw
# SHA256 from https://github.com/kyverno/chainsaw/releases/download/v0.2.13/checksums.txt
RUN curl -fL -o chainsaw.tar.gz \
      https://github.com/kyverno/chainsaw/releases/download/v0.2.13/chainsaw_linux_amd64.tar.gz \
    && echo "6c8d4cdccacbea7100a8354893b3176d874eecfe70c930fbe0496b7967d61ca4  chainsaw.tar.gz" | sha256sum -c - \
    && tar -xzf chainsaw.tar.gz chainsaw \
    && chmod +x chainsaw \
    && mv chainsaw /usr/local/bin/ \
    && rm chainsaw.tar.gz

# Install kubectl and oc (OCP 4.21.17)
# SHA256 from https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.21.17/sha256sum.txt
RUN curl -fL -o oc.tar.gz \
      https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.21.17/openshift-client-linux.tar.gz \
    && echo "c4adfe28fe0ea05eec931c39dd1867afc3d489589fbf12b2f27001f98cd77b95  oc.tar.gz" | sha256sum -c - \
    && tar -xzf oc.tar.gz kubectl oc \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/ \
    && rm oc.tar.gz

# Install AWS CLI v2.34.53
# SHA256 computed from https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.34.53.zip
# (AWS does not publish .sha256 sidecar files; PGP sig at https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.34.53.zip.sig)
RUN curl -fL -o awscliv2.zip \
      https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.34.53.zip \
    && echo "b06c9865123cf1bc6f8761a0bf3161630e6f55d84363f9fda89535e2cdbe654b  awscliv2.zip" | sha256sum -c - \
    && unzip awscliv2.zip \
    && ./aws/install \
    && rm -rf awscliv2.zip aws/

# Install Azure CLI 2.86.0 via APT repo (replaces curl|bash pipe)
# golang:1.25 base is Debian trixie; azure-cli apt repo has no trixie dist entry, bookworm packages work on trixie.
# GPG key SHA256: 2fa9c05d591a1582a9aba276272478c262e95ad00acf60eaee1644d93941e3c6
# If sha256sum fails here, Microsoft rotated their GPG key - update the hash and verify the new key.
RUN mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://packages.microsoft.com/keys/microsoft.asc -o /tmp/microsoft.asc \
    && echo "2fa9c05d591a1582a9aba276272478c262e95ad00acf60eaee1644d93941e3c6  /tmp/microsoft.asc" | sha256sum -c - \
    && gpg --dearmor < /tmp/microsoft.asc > /etc/apt/keyrings/microsoft.gpg \
    && chmod go+r /etc/apt/keyrings/microsoft.gpg \
    && rm /tmp/microsoft.asc \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/microsoft.gpg] https://packages.microsoft.com/repos/azure-cli bookworm main" \
      > /etc/apt/sources.list.d/azure-cli.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends azure-cli=2.86.0-1~bookworm \
    && rm -rf /var/lib/apt/lists/*

# Install Google Cloud CLI 569.0.0
# SHA256 computed from https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-569.0.0-linux-x86_64.tar.gz
# (Google does not publish .sha256 sidecar files)
RUN curl -fL -o google-cloud-cli.tar.gz \
      https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-569.0.0-linux-x86_64.tar.gz \
    && echo "05f2921b228228d75f34721792219c358373ea8c96a3294a374220ae558be82d  google-cloud-cli.tar.gz" | sha256sum -c - \
    && tar -xf google-cloud-cli.tar.gz \
    && ./google-cloud-sdk/install.sh -q \
    && rm google-cloud-cli.tar.gz

# Install IBM Cloud CLI 2.43.1 (replaces curl|bash pipe)
# SHA256 sourced from IBM metadata API and verified by download.
# IBM does not publish .sha256 sidecar files; recompute with:
#   curl -fsSL https://download.clis.cloud.ibm.com/ibm-cloud-cli-dn/2.43.1/IBM_Cloud_CLI_2.43.1_amd64.tar.gz | sha256sum
ENV IBM_CLI_VERSION=2.43.1
RUN curl -fsSL "https://download.clis.cloud.ibm.com/ibm-cloud-cli-dn/${IBM_CLI_VERSION}/IBM_Cloud_CLI_${IBM_CLI_VERSION}_amd64.tar.gz" \
      -o /tmp/ibmcli.tar.gz \
    && echo "9d6080c0b69f5e5cf960ef4e6e583aa240b80ce21c526c8b8149090f4821b18d  /tmp/ibmcli.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/ibmcli.tar.gz -C /tmp \
    && /tmp/Bluemix_CLI/install \
    && rm -rf /tmp/ibmcli.tar.gz /tmp/Bluemix_CLI

# Add gcloud to PATH
ENV PATH="/tmp/google-cloud-sdk/bin:${PATH}"

# Install Claude Code CLI via signed apt repository
RUN install -d -m 0755 /etc/apt/keyrings \
    && curl -fsSL https://downloads.claude.ai/keys/claude-code.asc \
         -o /etc/apt/keyrings/claude-code.asc \
    && echo "deb [signed-by=/etc/apt/keyrings/claude-code.asc] https://downloads.claude.ai/claude-code/apt/stable stable main" \
         > /etc/apt/sources.list.d/claude-code.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends claude-code \
    && rm -rf /var/lib/apt/lists/*

# Switch to non-root before the build step.
# chown /tmp/tempo-operator: COPY above ran as root; make build writes bin/ inside this dir.
# chown /tmp/google-cloud-sdk: install.sh writes update configs back here at runtime.
# HOME: golang base sets HOME=/root which UID 65532 cannot write to; az/gcloud/ibmcloud all need a writable home.
# IBMCLOUD_HOME: ibmcloud plugin install writes here at test runtime; explicit path avoids relying on $HOME resolution.
RUN useradd -u 65532 -m -d /home/tempo -s /bin/bash tempo \
    && chown -R tempo:tempo /tmp/tempo-operator /tmp/google-cloud-sdk /home/tempo
ENV HOME=/home/tempo
ENV IBMCLOUD_HOME=/home/tempo/.ibmcloud
# GOPATH must point to a user-writable directory; the golang base image sets GOPATH=/go (root-owned).
ENV GOPATH=/home/tempo/go
USER tempo

# Set the working directory
WORKDIR /tmp/tempo-operator

# Build required binaries
RUN make build
