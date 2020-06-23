# kafka-autoconnector

A K8s operator for managing the lifecycle of Kafka Connect connectors.

## Overview

The Kafka AutoConnector Operator provides Kubernetes native deployment and management of Confluent Kafka Connect connectors. The purpose of this project is to enable managing connectors the GitOps-way via Kubernetes CRDs.

Some of the features of the Kafka AutoConnector Operator include:

* Generic support for all types of connectors
* Health monitoring of managed connectors
* Auto-restarting of failed tasks and connectors
* Custom connectors metrics compatible with Prometheus

## Installation

In order to install the Kafka AutoConnector Operator, you should deploy the following resources in your Kubernetes cluster:

* deploy/service_account.yaml
* deploy/role_binding.yaml
* deploy/role.yaml
* deploy/operator.yaml
* deploy/crds/skynet.walmartdigital.cl_genericautoconnectors_crd.yaml

## Configuration

The following configuration settings can be controlled via environment variables:

* **KAFKA_CONNECT_ADDR**: The address of your Kafka Connect instance
* **CUSTOM_METRICS_PORT**: The port on which the custom metrics will be served in the operator container
* **CUSTOM_METRICS_PORT_NAME**: The name of the custom metrics port in the K8s *Service* that will be created automatically
* **SERVICE_MONITOR_LABELS**: The labels to apply to the custom metrics `ServiceMonitor` as a comma-separated list of key/value pairs, e.g., `name:kafka-autoconnector,release:prometheus,hello:world`. This might be relevant depending on your local Prometheus Operator configuration.

### Example Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kafka-autoconnector
spec:
  replicas: 1
  selector:
    matchLabels:
      name: kafka-autoconnector
  template:
    metadata:
      labels:
        name: kafka-autoconnector
    spec:
      serviceAccountName: kafka-autoconnector
      containers:
        - name: kafka-autoconnector
          image: kafka-autoconnector
          args:
            - '--zap-level=debug'
          command:
          - kafka-autoconnector
          env:
            - name: KAFKA_CONNECT_ADDR
              value: kafka-connect:8083
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "kafka-autoconnector"
            - name: CUSTOM_METRICS_PORT_NAME
              value: "foobar-metrics"
            - name: CUSTOM_METRICS_PORT
              value: "5555"
```

## How to Specify a Connector

Once the operator and the GenericAutoConnector CRD is installed, you can specify a connector as follows:

```yaml
apiVersion: skynet.walmartdigital.cl/v1alpha1
kind: GenericAutoConnector
metadata:
  name: example-connector
  namespace: default
spec:
  connector.config:
    connector.class: "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector"
    type.name: "log"
    topics: "dumblogger-logs,_ims.logs,_amida.logs,_osiris.logs,_midas.logs,_kimun.logs"
    topic.index.map: "dumblogger-logs:<logs-pd-dumblogger-{now/d}>,_ims.logs:<logs-pd-ims-{now/d}>,_amida.logs:<logs-pd-amida-{now/d}>,_osiris.logs:<logs-pd-osiris-{now/d}>,_midas.logs:<logs-pd-midas-{now/d}>,_kimun.logs:<logs-pd-kimun-{now/d}>"
    connection.url: "http://elasticsearch-master:9200"
    connection.username: "${vault:secret/underworld/kafka-connect/tools/elasticlogs:username}"
    connection.password: "${vault:secret/underworld/kafka-connect/tools/elasticlogs:password}"
    key.ignore: "true"
    schema.ignore: "true"
    behavior.on.malformed.documents: "ignore"
    batch.size: "200"
    max.in.flight.requests: "5"
    max.buffered.records: "20000"
    linger.ms: "1"
    flush.timeout.ms: "10000"
    max.retries: "5"
    retry.backoff.ms: "100"
    connection.compression: "false"
    connection.timeout.ms: "1000"
    read.timeout.ms: "3000"
    max.tasks: "2"
```

**Note**: Make sure all configuration parameters values are of type `string`, Kafka Connect will take care of converting these values to the appropriate data types when loading the configuration of the specified connector type.

## Metrics

The Kafka AutoConnector operator exposes 3 types of metrics by default:

* `kafka_autoconnector_total_connector_tasks`: The total number of tasks enabled on a connector.
* `kafka_autoconnector_running_connector_tasks`: The number of tasks in `RUNNING` state on a given connector.
* `kafka_autoconnector_connector_uptime`: The number of consecutive seconds that the connector has been in the `RUNNING` state. 

By default, the operator will install a K8s *Service* and the associated *ServiceMonitor* so as to instruct a local instance of Prometheus to scrape the aforementioned metrics.

## Building the Operator and Running it Locally

### Software Prerequisites

In order to run the Kafka AutoConnector Operator on your local machine, you will need the following tools:

* Minikube (tested on v1.8.2 and K8s v1.17.3)
* Skaffold (tested on v1.7.0)
* Helm 3 (tested on v3.1.2)

If you want to test the metrics feature or the operator's integration with Elasticsearch, you will need to manually add the corresponding Helm repos manually:

```
helm repo add elastic https://helm.elastic.co
helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo update
```

If you plan to run the operator with the full E2E stack, it is preferable to run Minikube with at least 8Gb of RAM and 4 CPUs. 

To build the Docker image for the operator, use the Operator SDK. From the root of the repo, run:
```
operator-sdk build kafka-autoconnector
```
To run the operator locally, use Skaffold. From the root of the project, run:
```
skaffold dev
```
