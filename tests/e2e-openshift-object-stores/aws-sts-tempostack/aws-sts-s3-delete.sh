#!/bin/bash
set -euo pipefail

# This script is meant to be run from OpenShift CI environment
# Run the script with ./aws-sts-s3-delete.sh TEMPO_NAME TEMPO_NAMESPACE

# Set the AWS credential file var
export AWS_SHARED_CREDENTIALS_FILE=$CLUSTER_PROFILE_DIR/.awscred
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
