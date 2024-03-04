#!/bin/bash

# Run the command and save its output
while true; do
    output=$(oc -n openshift-monitoring exec alertmanager-main-0 -- amtool --alertmanager.url http://localhost:9093 alert query SpanREDFrontendAPIRequestLatency 2>&1)

    # Check if the command was successful
    if [ $? -ne 0 ]; then
        echo "Error executing oc command: $output"
        exit 1
    fi

    # Check if the alert is active
    if echo "$output" | grep -q "SpanREDFrontendAPIRequestLatency.*active"; then
        echo "Alert SpanREDFrontendAPIRequestLatency is firing"
        exit 0
    else
        echo "Alert SpanREDFrontendAPIRequestLatency is not firing"
        sleep 5 # wait for 5 seconds before checking again
    fi
done
