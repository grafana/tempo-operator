#!/bin/bash
set -euo pipefail

PROJECT_ID=$(gcloud config get-value project)
SERVICE_ACCOUNT_NAME="ikanse-gcp-wif-tempo-sa"
GCS_KEY_FILE="/tmp/gcp-wif-tempo.json"
BUCKET_NAME="ikanse-gcp-wif-tempo"

# Create GCP service account
SERVICE_ACCOUNT_EMAIL=$(gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
    --display-name="TempoStack Account" \
    --project "$PROJECT_ID" \
    --format='value(email)' \
    --quiet)

# Wait for the service account to be ready
echo "Waiting for service account $SERVICE_ACCOUNT_EMAIL to be ready..."
MAX_RETRIES=10
RETRY_COUNT=0
while ! gcloud iam service-accounts describe "$SERVICE_ACCOUNT_EMAIL" --project "$PROJECT_ID" &> /dev/null; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Service account $SERVICE_ACCOUNT_EMAIL not found after $MAX_RETRIES retries. Exiting."
        exit 1
    fi
    echo "Service account not yet available. Retrying in 5 seconds..."
    sleep 5
    RETRY_COUNT=$((RETRY_COUNT + 1))
done
echo "Service account $SERVICE_ACCOUNT_EMAIL is ready."

# Set the GCP and TempoStack vars
TEMPO_NAME="gcpwiftm"
TEMPO_NAMESPACE="chainsaw-gcpwif-tempo"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')

# Bind the required GCP roles to the created SA at the project level
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" \
  --format=none \
  --quiet

# Workload Identity Bindings: Allow Kubernetes Service Accounts to impersonate the Google Service Account
gcloud iam service-accounts add-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --project="$PROJECT_ID" \
  --quiet

gcloud iam service-accounts add-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
  --project="$PROJECT_ID" \
  --quiet

# Get provider ID from GCP
PROVIDER_ID=$(gcloud iam workload-identity-pools providers list \
  --project="$PROJECT_ID" \
  --location="global" \
  --workload-identity-pool="$POOL_ID" \
  --filter="displayName:$POOL_ID" \
  --format="value(name)" | awk -F'/' '{print $NF}')

# Create a credentials configuration file for the managed identity to be used by TempoStack
gcloud iam workload-identity-pools create-cred-config \
    "projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/providers/$PROVIDER_ID" \
    --service-account="$SERVICE_ACCOUNT_EMAIL" \
    --credential-source-file=/var/run/secrets/storage/serviceaccount/token \
    --credential-source-type=text \
    --output-file="$GCS_KEY_FILE"

echo "Checking if bucket $BUCKET_NAME exists..."
if gsutil ls "gs://$BUCKET_NAME" > /dev/null 2>&1; then
    echo "Bucket $BUCKET_NAME found. Attempting to remove..."
    gcloud alpha storage rm --recursive "gs://$BUCKET_NAME"
    if [ $? -ne 0 ]; then
        echo "Failed to remove bucket $BUCKET_NAME."
        exit 1
    fi
    echo "Bucket $BUCKET_NAME removed successfully."
else
    echo "Bucket $BUCKET_NAME does not exist (as expected). Proceeding to create."
fi

echo "Waiting for the bucket to be confirmed deleted (if it was removed)."
BUCKET_DELETION_RETRIES=6
DELETE_RETRY_COUNT=0
while gsutil ls "gs://$BUCKET_NAME" > /dev/null 2>&1 && [ $DELETE_RETRY_COUNT -lt $BUCKET_DELETION_RETRIES ]; do
  echo "Bucket $BUCKET_NAME still detected. Waiting 5 seconds for deletion..."
  sleep 5
  DELETE_RETRY_COUNT=$((DELETE_RETRY_COUNT + 1))
done

if [ $DELETE_RETRY_COUNT -ge $BUCKET_DELETION_RETRIES ]; then
    echo "Warning: Bucket $BUCKET_NAME still exists after waiting period. This might cause issues for creation."
fi

echo "Attempting to create a new bucket: gs://$BUCKET_NAME in us-central1..."
gsutil mb -l us-central1 -p "$PROJECT_ID" "gs://$BUCKET_NAME"
if [ $? -ne 0 ]; then
    echo "Failed to create bucket $BUCKET_NAME."
    exit 1
fi
echo "Bucket $BUCKET_NAME created successfully."

echo "Grant access to the bucket by service account."
gcloud storage buckets add-iam-policy-binding "gs://$BUCKET_NAME" \
    --role="roles/storage.admin" \
    --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
    --condition=None

PROVIDER_NAME="projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/providers/$PROVIDER_ID"
AUDIENCE=$(gcloud iam workload-identity-pools providers describe "$PROVIDER_NAME" --format='value(oidc.allowedAudiences[0])')

# Create Kubernetes secret to be used with TempoStack
kubectl -n "$TEMPO_NAMESPACE" create secret generic gcs-secret \
  --from-literal=bucketname="$BUCKET_NAME" \
  --from-literal=audience="$AUDIENCE" \
  --from-file=key.json="$GCS_KEY_FILE"
if [ $? -ne 0 ]; then
  echo "Failed to create secret"
  exit 1
fi