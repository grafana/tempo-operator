# Metrics Generator
```
kubectl apply -f metricsgenerator.yaml
kubectl port-forward svc/grafana 3000:3000
```

Open [Grafana](http://localhost:3000/explore).
Navigate to "Service Graph" for span metrics and service graph.
