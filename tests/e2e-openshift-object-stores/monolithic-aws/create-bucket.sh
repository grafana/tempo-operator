#!/bin/bash

# Define constants
BUCKET_NAME="ikanse-monolithic-s3"
REGION="us-east-2"
AWS_BUCKET_ENDPOINT="https://s3.${REGION}.amazonaws.com"

# Fetch AWS credentials based on CLUSTER_PROFILE_DIR variable
if [ -n "${CLUSTER_PROFILE_DIR}" ]; then
    export AWS_ACCESS_KEY_ID=$(grep "aws_access_key_id=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
    export AWS_SECRET_ACCESS_KEY=$(grep "aws_secret_access_key=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
else
    export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_access_key_id' | base64 -d)
    if [ $? -ne 0 ]; then
        echo "Failed to fetch AWS_ACCESS_KEY_ID"
        exit 1
    fi

    export AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_secret_access_key' | base64 -d)
    if [ $? -ne 0 ]; then
        echo "Failed to fetch AWS_SECRET_ACCESS_KEY"
        exit 1
    fi
fi

# Check if the bucket exists
if aws s3api head-bucket --bucket $BUCKET_NAME --region $REGION 2>/dev/null; then
    # Remove the bucket if it exists
    aws s3 rb s3://$BUCKET_NAME --region $REGION --force
    if [ $? -ne 0 ]; then
        echo "Failed to remove bucket"
        exit 1
    fi

    # Check if the bucket still exists
    if aws s3api head-bucket --bucket $BUCKET_NAME --region $REGION 2>/dev/null; then
        echo "Bucket still exists after deletion, wait for 30 seconds."
        sleep 30
    fi
fi

# Create a new bucket
aws s3api create-bucket --bucket $BUCKET_NAME --region $REGION --create-bucket-configuration LocationConstraint=$REGION
if [ $? -ne 0 ]; then
    echo "Failed to create bucket"
    exit 1
fi

# Create a new secret
kubectl -n $NAMESPACE create secret generic s3-secret \
  --from-literal=bucket="$BUCKET_NAME" \
  --from-literal=endpoint="$AWS_BUCKET_ENDPOINT" \
  --from-literal=access_key_id="$AWS_ACCESS_KEY_ID" \
  --from-literal=access_key_secret="$AWS_SECRET_ACCESS_KEY"
if [ $? -ne 0 ]; then
    echo "Failed to create secret"
    exit 1
fi

echo "Script executed successfully"
