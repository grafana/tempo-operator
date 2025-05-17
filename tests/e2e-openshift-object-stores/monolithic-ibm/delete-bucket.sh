#!/bin/bash

# List objects in the bucket
ibmcloud cos objects --bucket ikanse-tempo-bucket-mono

# Delete all objects in the bucket
ibmcloud cos list-objects --bucket ikanse-tempo-bucket-mono --output json | jq -r '.Contents[].Key' | xargs -I {} ibmcloud cos object-delete --bucket ikanse-tempo-bucket-mono --force --key {}

# Delete the bucket
ibmcloud cos bucket-delete --bucket ikanse-tempo-bucket-mono --force

# Delete the service key
ibmcloud resource service-key-delete ikanse-tempo-bucket-mono --force

# Delete the service instance
ibmcloud resource service-instance-delete ikanse-tempo-bucket-mono --force

# Delete the resource group
ibmcloud resource group-delete ikanse-tracing-mono --force
