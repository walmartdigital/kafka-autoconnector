# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Support for specifying custom metrics Service and ServiceMonitor labels as environment variables

## [0.1.10] - 2020-06-23

### Added
- Config package to manage operator configuration centrally
- Creation of custom metrics K8s Service and ServiceMonitor by default
- Installation of Prometheus operator with Helm as part of stack for E2E tests
- Support for parameterizing custom metrics port and port name
- Necessary labels to service and service monitor to be picked up by Prometheus
- README
- CHANGELOG

### Fixed
- Typo in names of custom metrics

### Removed
- Unused check.sh script

## [0.1.9] - 2020-06-19

### Added
- Logic to handle 409 HTTP response code from Kafka Connect when reading and updating connectors
### Changed
- Improved log message when deleting a connector
- Version of go-kaya dependency (v0.1.0)
### Removed
- Support for ESSinkConnector API/CRD

## [0.1.8] - 2020-06-15

This release has a breaking changes for the `ESSinkConnector` API/CRD. The ESSinkConnectorConfig field is no longer a custom data structure but rather a `map[string]string`.

### Added
- Fully functional GenericAutoConnector API and controller
- Connector uptime metric
### Changed
- Improved error message when KafkaConnect client fails to Read connector
- ESSinkConnectorConfig to be a `map[string]string`

## [0.1.7] - 2020-06-10

### Added
- Basic connector metrics (total number of tasks and number of runnning tasks)
### Fixed
- Docker image tag generation logic (in `build.sh`) to account for when git tags exist

## [0.1.3] - 2020-06-04

### Fixed
- Golang build command to create statically-linked binary

## [0.1.1] - 2020-06-04

### Changed
- Docker repo to walmartdigital/kafka-autoconnector

## [0.1.0] - 2020-06-03

### Added
- Support for ESSInkConnector for managing Elasticsearch Sink connectors
- Logic for restarting failed tasks and connectors as well as hard-resetting connectors (i.e., delete and recreate)