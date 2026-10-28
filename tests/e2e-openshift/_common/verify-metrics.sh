#!/bin/bash
set -euo pipefail

: "${NAMESPACE:?Chainsaw must supply the metrics-reader namespace}"
if [[ "$#" -eq 0 ]]; then
    echo "Usage: $0 promql [promql ...]" >&2
    exit 1
fi

TOKEN=$(oc create token e2e-metrics-reader -n "$NAMESPACE")
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o jsonpath='{.spec.host}')

for query in "$@"; do
    # The enclosing Chainsaw operation supplies the timeout.
    while true; do
        response=$(curl --fail --silent --show-error -k -G \
            -H "Authorization: Bearer $TOKEN" \
            --data-urlencode "query=$query" \
            "https://$THANOS_QUERIER_HOST/api/v1/query")
        if ! jq -e '.status == "success"' <<< "$response" > /dev/null; then
            echo "Prometheus query failed: $response" >&2
            exit 1
        fi

        count=$(jq -r '.data.result | length' <<< "$response")
        if [[ "$count" -gt 0 ]]; then
            echo "Query '$query' returned results."
            break
        fi
        echo "Query '$query' returned no results. Retrying..."
        sleep 5
    done
done
