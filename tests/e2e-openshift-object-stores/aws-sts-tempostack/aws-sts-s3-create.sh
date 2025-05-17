#!/bin/bash
set -euo pipefail

# This script is meant to be run from OpenShift CI environment
# Run the script with ./aws-sts-s3-create.sh TEMPO_NAME TEMPO_NAMESPACE

# Check if OPENSHIFT_BUILD_NAMESPACE is unset or empty
if [ -z "${OPENSHIFT_BUILD_NAMESPACE+x}" ]; then
    OPENSHIFT_BUILD_NAMESPACE="cioptmstack"
    export OPENSHIFT_BUILD_NAMESPACE
fi

echo "OPENSHIFT_BUILD_NAMESPACE is set to: $OPENSHIFT_BUILD_NAMESPACE"

if [ -z "${CLUSTER_PROFILE_DIR+x}" ]; then
    echo "Warning: CLUSTER_PROFILE_DIR is not set, proceeding without it..."
else
    export AWS_SHARED_CREDENTIALS_FILE="$CLUSTER_PROFILE_DIR/.awscred"
    echo "AWS_SHARED_CREDENTIALS_FILE is set to: $AWS_SHARED_CREDENTIALS_FILE"
fi

export AWS_PAGER=""
region=us-east-2
tempo_name="$1"
tempo_ns="$2"
bucket_name=tracing-$tempo_ns-$OPENSHIFT_BUILD_NAMESPACE

# Create a S3 bucket 
aws s3api create-bucket --bucket $bucket_name --region $region --create-bucket-configuration LocationConstraint=$region

# Set required vars to create AWS IAM policy and role
oidc_provider=$(oc get authentication cluster -o json | jq -r '.spec.serviceAccountIssuer' | sed 's~http[s]*://~~g')
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
cluster_id=$(oc get clusterversion -o jsonpath='{.items[].spec.clusterID}{"\n"}')
trust_rel_file="/tmp/$tempo_ns-trust.json"
role_name="tracing-$tempo_ns-$OPENSHIFT_BUILD_NAMESPACE"

# Create a trust relationship file
cat > "$trust_rel_file" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${aws_account_id}:oidc-provider/${oidc_provider}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${oidc_provider}:sub": [
            "system:serviceaccount:${tempo_ns}:tempo-${tempo_name}",
            "system:serviceaccount:${tempo_ns}:tempo-${tempo_name}-query-frontend"
         ]
       }
     }
   }
 ]
}
EOF

echo "Creating IAM role '$role_name'..."
role_arn=$(aws iam create-role \
             --role-name "$role_name" \
             --assume-role-policy-document "file://$trust_rel_file" \
             --query Role.Arn \
             --output text)

echo "Attaching role policy 'AmazonS3FullAccess' to role '$role_name'..."
aws iam attach-role-policy \
  --role-name "$role_name" \
  --policy-arn "arn:aws:iam::aws:policy/AmazonS3FullAccess"

echo "Role created and policy attached successfully!"

echo "Create the secret to be used with Tempo"
oc -n $tempo_ns create secret generic aws-sts \
  --from-literal=bucket="$bucket_name" \
  --from-literal=region="$region" \
  --from-literal=role_arn="$role_arn"
