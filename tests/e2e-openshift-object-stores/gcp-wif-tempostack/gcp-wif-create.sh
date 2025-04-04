#!/bin/bash
set -euo pipefail

PROJECT_ID=$(gcloud config get-value project)
SERVICE_ACCOUNT_NAME="ikanse-gcp-wif-tempo-sa"

# Create GCP service account
SERVICE_ACCOUNT_EMAIL=$(gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
    --display-name="TempoStack Account" \
    --project "$PROJECT_ID" \
    --format='value(email)' \
    --quiet)

# Bind the required GCP roles to the created SA
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser"\
  --format=none \
  --quiet

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" \
  --format=none \
  --quiet

# Set the GCP and TempoStack vars
TEMPO_NAME="gcpwiftm"
TEMPO_NAMESPACE="chainsaw-gcpwif-tempo"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')

# Bind the required roles to TempoStack service accounts
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --quiet

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --quiet

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
  --quiet

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
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
    --credential-source-file=/var/run/secrets/kubernetes.io/serviceaccount/token \
    --credential-source-type=text \
    --output-file="/tmp/gcp-wif-tempo.json"

GCS_KEY_FILE="/tmp/gcp-wif-tempo.json"
BUCKET_NAME="ikanse-gcp-wif-tempo"

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
gsutil mb -l us-central1 -p "$PROJECT_ID" gs://$BUCKET_NAME
if [ $? -ne 0 ]; then
    echo "Failed to create bucket"
    exit 1
fi

PROVIDER_NAME="projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/providers/$PROVIDER_ID"
AUDIENCE=$(gcloud iam workload-identity-pools providers describe "$PROVIDER_NAME" --format='value(oidc.allowedAudiences[0])')

# Create Kubernetes secret to be used with TempoStack
kubectl -n "$TEMPO_NAMESPACE" create secret generic gcs-secret \
  --from-literal=bucketname="$BUCKET_NAME" \
  --from-literal=iam_sa="$SERVICE_ACCOUNT_NAME" \
  --from-literal=iam_sa_project_id="$PROJECT_ID" \
  --from-file=key.json="$GCS_KEY_FILE"
if [ $? -ne 0 ]; then
  echo "Failed to create secret"
  exit 1
fi