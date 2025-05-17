#!/bin/bash

# Set the IBM Cloud credential
export IBMCLOUD_API_KEY=$(oc get secrets qe-ibmcloud-creds -n kube-system -o json | jq -r '.data.apiKey' | base64 -d)

# Login to IBM Cloud
ibmcloud login -r us-east

# Install the required plugins
ibmcloud plugin install -f cloud-object-storage
ibmcloud plugin install -f infrastructure-service

# Create a resource group
ibmcloud resource group-create ikanse-tracing
ibmcloud target -g ikanse-tracing

# Create a service instance for the object store
ibmcloud resource service-instance-create ikanse-tempo-bucket cloud-object-storage standard global -d premium-global-deployment

IBM_SERVICE_INSTANCE_ID=$(ibmcloud resource service-instance ikanse-tempo-bucket -o json | jq -r '.[0].crn')

# Create the object store bucket
ibmcloud cos bucket-create --bucket ikanse-tempo-bucket --ibm-service-instance-id "$IBM_SERVICE_INSTANCE_ID"

# Create service key for connecting to the bucket
ibmcloud resource service-key-create ikanse-tempo-bucket Writer --instance-name ikanse-tempo-bucket --parameters '{"HMAC":true}'

# Create secret in OpenShift with IBM cloud bucket credentials
json_output=$(ibmcloud resource service-key ikanse-tempo-bucket -o json)
export IBM_BUCKET_ACCESS_KEY=$(echo $json_output | jq -r '.[0].credentials.cos_hmac_keys.access_key_id')
export IBM_BUCKET_SECRET_KEY=$(echo $json_output | jq -r '.[0].credentials.cos_hmac_keys.secret_access_key')
export IBM_BUCKET_ENDPOINT="https://s3.us-east.cloud-object-storage.appdomain.cloud"

kubectl -n $NAMESPACE create secret generic ibm-cos-secret \
  --from-literal=bucket="ikanse-tempo-bucket" \
  --from-literal=endpoint="$IBM_BUCKET_ENDPOINT" \
  --from-literal=access_key_id="$IBM_BUCKET_ACCESS_KEY" \
  --from-literal=access_key_secret="$IBM_BUCKET_SECRET_KEY"
