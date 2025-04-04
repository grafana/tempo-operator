#!/bin/bash
set -uo pipefail

PROJECT_ID=$(gcloud config get-value project)
SERVICE_ACCOUNT_NAME="ikanse-gcp-wif-tempo-sa"
TEMPO_NAME="gcpwiftm"
TEMPO_NAMESPACE="chainsaw-gcpwif-tempo"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')
BUCKET_NAME="ikanse-gcp-wif-tempo"
GCS_KEY_FILE="/tmp/gcp-wif-tempo.json"

# Fetch the service account email using the name
SERVICE_ACCOUNT_EMAIL=$(gcloud iam service-accounts list \
  --project="$PROJECT_ID" \
  --filter="displayName:TempoStack Account" \
  --format='value(email)')

if [ -z "$SERVICE_ACCOUNT_EMAIL" ]; then
  echo "Error: Service account with display name 'TempoStack Account' not found. Cannot proceed with cleanup."
  exit 1
fi

echo "Starting cleanup (excluding Workload Identity Pool Provider)..."
echo "Target Service Account Email: **$SERVICE_ACCOUNT_EMAIL**"

echo "Removing bucket-level IAM policy binding for service account '**$SERVICE_ACCOUNT_EMAIL**' on bucket '**$BUCKET_NAME**'..."
gcloud storage buckets remove-iam-policy-binding "gs://$BUCKET_NAME" \
    --role="roles/storage.admin" \
    --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
    --quiet
echo "Bucket-level IAM policy binding removed."

# Remove IAM policy bindings for the created service account (project level)
echo "Removing project-level IAM policy bindings for service account '**$SERVICE_ACCOUNT_EMAIL**'..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/storage.objectAdmin" --quiet
echo "Project-level IAM policy bindings for service account '**$SERVICE_ACCOUNT_EMAIL**' removed."

# Remove IAM policy bindings for TempoStack service accounts (Workload Identity Principals)
# These were added directly to the Google Service Account's IAM policy
echo "Removing Workload Identity bindings from Google Service Account '**$SERVICE_ACCOUNT_EMAIL**'..."

# tempo-${TEMPO_NAME} bindings
gcloud iam service-accounts remove-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}" \
  --project="$PROJECT_ID" \
  --quiet

# tempo-${TEMPO_NAME}-query-frontend bindings
gcloud iam service-accounts remove-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend" \
  --project="$PROJECT_ID" \
  --quiet
echo "Workload Identity bindings removed from Google Service Account '**$SERVICE_ACCOUNT_EMAIL**'."

# Delete the GCP service account
echo "Deleting service account '**$SERVICE_ACCOUNT_EMAIL**'..."
gcloud iam service-accounts delete "$SERVICE_ACCOUNT_EMAIL" --quiet
if [ $? -eq 0 ]; then
  echo "Service account '**$SERVICE_ACCOUNT_EMAIL**' deleted successfully."
else
  echo "Failed to delete service account '**$SERVICE_ACCOUNT_EMAIL**'."
fi

# Remove GCS bucket
echo "Deleting GCS bucket '**$BUCKET_NAME**'..."
gcloud alpha storage rm --recursive gs://"$BUCKET_NAME" --quiet
if [ $? -eq 0 ]; then
  echo "GCS bucket '**$BUCKET_NAME**' deleted successfully."
else
  echo "Failed to delete GCS bucket '**$BUCKET_NAME**'."
fi

echo "Deleting **gcs-secret** Kubernetes secret..."
if kubectl get secret gcs-secret -n "$TEMPO_NAMESPACE" &> /dev/null; then
  kubectl delete secret gcs-secret -n "$TEMPO_NAMESPACE" --ignore-not-found=true
  echo "Kubernetes secret '**gcs-secret**' deleted."
else
  echo "Kubernetes secret '**gcs-secret**' not found in namespace '**$TEMPO_NAMESPACE**'. Skipping deletion."
fi

echo "Delete the GCS keyfile"
rm "$GCS_KEY_FILE"

echo "Cleanup completed."