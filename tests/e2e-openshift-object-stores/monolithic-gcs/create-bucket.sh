#!/bin/bash

# Define constants
BUCKET_NAME="ikanse-monolithic-gcs"

oc extract -n kube-system secret/gcp-credentials --to=/tmp --confirm
if [ $? -ne 0 ]; then
    echo "Failed to fetch GCS service account json."
    exit 1
fi
GCS_KEY_FILE="/tmp/service_account.json"

# Check if the bucket exists
gsutil ls gs://$BUCKET_NAME > /dev/null 2>&1
if [ $? -eq 0 ]; then
    # Remove the bucket if it exists
    gcloud alpha storage rm --recursive gs://$BUCKET_NAME
    if [ $? -ne 0 ]; then
        echo "Failed to remove bucket"
        exit 1
    fi
fi

# Wait for the bucket to be deleted for 30 seconds.
gsutil ls gs://$BUCKET_NAME > /dev/null 2>&1
if [ $? -eq 0 ]; then
  sleep 30
fi

# Create a new bucket
gsutil mb gs://$BUCKET_NAME
if [ $? -ne 0 ]; then
    echo "Failed to create bucket"
    exit 1
fi

# Create a new secret
kubectl -n $NAMESPACE create secret generic gcs-secret \
  --from-literal=bucketname="$BUCKET_NAME" \
  --from-file=key.json="$GCS_KEY_FILE"
if [ $? -ne 0 ]; then
    echo "Failed to create secret"
    exit 1
fi

echo "Script executed successfully"
