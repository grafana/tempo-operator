# Runbook
This document should help with remediation of operational issues in Tempo Operator.

## TempoStackUnhealthy
Check the `Message` field of the status condition with `Status: True` of the affected TempoStack instance for information on how to resolve this issue:
```
kubectl -n <namespace> describe tempo <instance>
```

## TempoOperatorReconcileError
The Operator failed to reconcile its managed resources. This leads to managed resources to be out of sync with the desired state.
The cause of this error can be various configuration errors, for example insufficient permissions.
To remediate this issue, please inspect the logs of the tempo operator pod:
```
kubectl -n <operator_namespace> logs deployment/tempo-operator-controller
```

## TempoOperatorReconcileDurationHigh
The Operator requires longer than 10 minutes to reconcile its managed resources.
Please inspect the logs of the tempo operator pod to find the root cause:
```
kubectl -n <operator_namespace> logs deployment/tempo-operator-controller
```
