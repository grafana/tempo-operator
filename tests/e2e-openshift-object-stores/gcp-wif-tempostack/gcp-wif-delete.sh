#!/bin/bash
set -euo pipefail

PROJECT_ID=$(gcloud config get-value project)
SERVICE_ACCOUNT_NAME="ikanse-gcp-wif-tempo-sa"
TEMPO_NAME="gcpwiftm"
TEMPO_NAMESPACE="chainsaw-gcpwif-tempo"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')
BUCKET_NAME="ikanse-gcp-wif-tempo"

# Fetch the service account email using the name
SERVICE_ACCOUNT_EMAIL=$(gcloud iam service-accounts list \
  --project="$PROJECT_ID" \
  --filter="displayName:TempoStack Account" \
  --format='value(email)')

if [ -z "$SERVICE_ACCOUNT_EMAIL" ]; then
  echo "Error: Service account with display name 'TempoStack Account' not found."
  exit 1
fi

echo "Starting cleanup (excluding Workload Identity Pool Provider)..."
echo "Target Service Account Email: $SERVICE_ACCOUNT_EMAIL"

# Remove GCS bucket
echo "Deleting GCS bucket '$BUCKET_NAME'..."
gcloud alpha storage rm --recursive gs://$BUCKET_NAME --quiet
if [ $? -eq 0 ]; then
  echo "GCS bucket '$BUCKET_NAME' deleted successfully."
else
  echo "Failed to delete GCS bucket '$BUCKET_NAME'."
fi

# Remove IAM policy bindings for TempoStack service accounts
echo "Removing IAM policy bindings for TempoStack service accounts..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --role="roles/iam.workloadIdentityUser" --quiet

gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --role="roles/storage.objectAdmin" --quiet

gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
  --role="roles/iam.workloadIdentityUser" --quiet

gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
  --role="roles/storage.objectAdmin" --quiet
echo "IAM policy bindings for TempoStack service accounts removed."

# Remove IAM policy bindings for the created service account
echo "Removing IAM policy bindings for service account '$SERVICE_ACCOUNT_EMAIL'..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" --quiet

gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" --quiet
echo "IAM policy bindings for service account '$SERVICE_ACCOUNT_EMAIL' removed."

# Delete the GCP service account
echo "Deleting service account '$SERVICE_ACCOUNT_EMAIL'..."
gcloud iam service-accounts delete "$SERVICE_ACCOUNT_EMAIL" --quiet
if [ $? -eq 0 ]; then
  echo "Service account '$SERVICE_ACCOUNT_EMAIL' deleted successfully."
else
  echo "Failed to delete service account '$SERVICE_ACCOUNT_EMAIL'."
fi

echo "Cleanup completed"