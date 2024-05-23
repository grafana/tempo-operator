#!/bin/bash

TOKEN=$(oc create token prometheus-user-workload -n openshift-user-workload-monitoring)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

#Check metrics used in the prometheus rules created for TempoStack. Refer issue https://issues.redhat.com/browse/TRACING-3399 for skipped metrics.
metrics="tempo_request_duration_seconds_count tempo_request_duration_seconds_sum tempo_request_duration_seconds_bucket tempo_build_info tempo_ingester_bytes_received_total tempo_ingester_flush_failed_retries_total tempo_ingester_failed_flushes_total tempo_ring_members"

for metric in $metrics; do
query="$metric"
count=0

# Keep fetching and checking the metrics until metrics with value is present.
while [[ $count -eq 0 ]]; do
    response=$(curl -k -H "Authorization: Bearer $TOKEN" -H "Content-type: application/json" "https://$THANOS_QUERIER_HOST/api/v1/query?query=$query")
    count=$(echo "$response" | jq -r '.data.result | length')

    if [[ $count -eq 0 ]]; then
    echo "No metric '$metric' with value present. Retrying..."
    sleep 5  # Wait for 5 seconds before retrying
    else
    echo "Metric '$metric' with value is present."
    fi
  done
done
