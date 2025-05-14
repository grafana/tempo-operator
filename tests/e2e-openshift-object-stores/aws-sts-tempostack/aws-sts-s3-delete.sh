#!/bin/bash
set -euo pipefail

# This script is meant to be run from OpenShift CI environment
# Run the script with ./aws-sts-s3-delete.sh TEMPO_NAME TEMPO_NAMESPACE

# Check if OPENSHIFT_BUILD_NAMESPACE is unset or empty
if [ -z "${OPENSHIFT_BUILD_NAMESPACE+x}" ]; then
    OPENSHIFT_BUILD_NAMESPACE="cioptmstack"
    export OPENSHIFT_BUILD_NAMESPACE
fi

echo "OPENSHIFT_BUILD_NAMESPACE is set to: $OPENSHIFT_BUILD_NAMESPACE"

if [ -z "${CLUSTER_PROFILE_DIR+x}" ]; then
    echo "Warning: CLUSTER_PROFILE_DIR is not set, proceeding without it..."
else
    export AWS_SHARED_CREDENTIALS_FILE="$CLUSTER_PROFILE_DIR/.awscred"
    echo "AWS_SHARED_CREDENTIALS_FILE is set to: $AWS_SHARED_CREDENTIALS_FILE"
fi

export AWS_PAGER=""
region=us-east-2
tempo_name="$1"
tempo_ns="$2"
bucket_name="tracing-$tempo_ns-$OPENSHIFT_BUILD_NAMESPACE"
role_name="tracing-$tempo_ns-$OPENSHIFT_BUILD_NAMESPACE"

# Remove role policy
echo "Remove IAM policy and delete role $role_name"
aws iam detach-role-policy --role-name "$role_name" \
--policy-arn "arn:aws:iam::aws:policy/AmazonS3FullAccess"

# Delete the IAM role
aws iam delete-role --role-name "$role_name"

# Delete the S3 bucket
aws s3 rb "s3://$bucket_name" --region $region --force
