# TLS Overview

## compactor, ingester, querier
Port  | Service               | TLS Enabled                     | TLS Certificate     | Verify Client Cert
----- | --------------------- | ------------------------------- | ------------------- | ------------------
3200  | Tempo API (HTTP)      | `featureGates.httpEncryption`   | internal            | yes
3101  | Tempo Internal (HTTP) | `featureGates.httpEncryption`   | internal            | no

## distributor
Port  | Service               | TLS Enabled                     | TLS Certificate     | Verify Client Cert
----- | --------------------- | ------------------------------- | ------------------- | ------------------
3200  | Tempo API (HTTP)      | `featureGates.httpEncryption`   | internal            | yes
3101  | Tempo Internal (HTTP) | `featureGates.httpEncryption`   | internal            | no
4317  | OTLP/gRPC             | `spec.template.distributor.tls` | custom              | no
4318  | OTLP/HTTP             | `spec.template.distributor.tls` | custom              | no
14268 | jaeger/thrift http    | `spec.template.distributor.tls` | custom              | no
6831  | jaeger/thrift compact | no                              | -                   | -
6832  | jaeger/thrift binary  | no                              | -                   | -
14250 | jaeger/grpc           | `spec.template.distributor.tls` | custom              | no
9411  | zipkin                | `spec.template.distributor.tls` | custom              | no

## query-frontend
Port  | Service               | TLS Enabled                     | TLS Certificate     | Verify Client Cert
----- | --------------------- | ------------------------------- | ------------------- | ------------------
3200  | Tempo API (HTTP)      | if `httpEncryption` and gateway | internal            | yes
3101  | Tempo Internal (HTTP) | `featureGates.httpEncryption`   | internal            | no
16686 | Jaeger UI (HTTP)      | if `httpEncryption` and gateway | internal            | yes
16685 | Jaeger UI (gRPC)      | if `httpEncryption` and gateway | internal            | yes

## gateway
Port  | Service               | TLS Enabled                     | TLS Certificate     | Verify Client Cert
----- | --------------------- | ------------------------------- | ------------------- | ------------------
8080  | public (HTTP)         | `servingCertsService`           | service-ca-operator | no
8090  | public (gRPC)         | `servingCertsService`           | service-ca-operator | no
8081  | internal (HTTP)       | `featureGates.httpEncryption`   | internal            | no

## TLS Clients
Client | TLS Enabled        | TLS Certificate | Notes
------ | ------------------ | --------------- | -----
S3     | `spec.storage.tls` | custom          | only custom CA is supported
Azure  | no                 | -               |
GCP    | no                 | -               |
