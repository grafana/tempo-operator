#!/bin/bash

# List objects in the bucket
ibmcloud cos objects --bucket ikanse-tempo-bucket

# Delete all objects in the bucket
ibmcloud cos list-objects --bucket ikanse-tempo-bucket --output json | jq -r '.Contents[].Key' | xargs -I {} ibmcloud cos object-delete --bucket ikanse-tempo-bucket --force --key {}

# Delete the bucket
ibmcloud cos bucket-delete --bucket ikanse-tempo-bucket --force

# Delete the service key
ibmcloud resource service-key-delete ikanse-tempo-bucket --force

# Delete the service instance
ibmcloud resource service-instance-delete ikanse-tempo-bucket --force

# Delete the resource group
ibmcloud resource group-delete ikanse-tracing --force
