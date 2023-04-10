// MIT License
//
// Copyright (c) 2023 Alexandre Mahdhaoui
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package main provides a program that periodically collects metrics about Amazon RDS clusters and instances and
// exports them in Prometheus format. It uses the AWS SDK for Go and the Prometheus Go client library to perform these
// operations.
//
// The program reads two environment variables: EXPORTER_AWS_API & INTERVAL_SECONDS, which specifies the time interval
// in seconds for fetching the data, and EXPORTER_SERVER_PORT, which specifies the port number for serving the
// Prometheus metrics.
//
// The program defines two main types: Config, which holds the AWS RDS API client, and Metrics, which holds the
// Prometheus metrics. The program also defines a struct RDSInfo to represent information about an Amazon RDS cluster.
//
// The main() function initializes the program by setting up the configuration, metrics, and HTTP server, and then
// starts a goroutine that periodically fetches RDS cluster and instance data and exports the metrics. The goroutine
// uses the snapshot() function to fetch the data and export the metrics.
//
// The snapshot() function fetches RDS cluster and instance data, merges them into a single slice of RDSInfos, and
// then exports the metrics for each RDSInfo. If any error occurs during the metric exporting process, the function
// will skip the problematic RDSInfo and continue exporting other RDSInfos.
//
// The export() function collects RDS info and validates its engine version against a map of allowed engine versions.
// If the version is deprecated, it will set the deprecatedGauge Prometheus metric to 1 and the availableGauge metric
// to 0, and vice versa if the version is available.
//
// The program also defines two helper functions: getEnvInteger() to read integer environment variables, and
// initHttpServer() to initialize the HTTP server.
package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	AwsApiIntervalEnvName = "EXPORTER_AWS_API_INTERVAL_SECONDS"
	ServerPortEnvName     = "EXPORTER_SERVER_PORT"
)

// Config holds the AWS RDS API client used to make calls to the Amazon RDS API.
// The NewConfig function creates a new Config struct with a pre-initialized RDSAPI client. The client is created with
// the AWS session shared configuration state enabled. If the AWS session shared configuration cannot be enabled, the
// function will panic.
type Config struct {
	RDS rdsiface.RDSAPI
}

// NewConfig creates and returns a new Config struct with a pre-initialized RDSAPI client.
// The client is created with the AWS session shared configuration state enabled.
// If the AWS session shared configuration cannot be enabled, the function will panic.
// The returned Config struct can be used to make calls to the Amazon RDS API.
func NewConfig() *Config {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return &Config{
		RDS: rds.New(sess),
	}
}

// Metrics defined to hold two Prometheus GaugeVecs, one for instances whose engine version is available, and the other
// for those whose version is deprecated. These metrics are initialized using the NewGaugeVec function of the prometheus
// package, and they include a namespace, subsystem, name, help string, and label names.
type Metrics struct {
	AvailableGauge  *prometheus.GaugeVec
	DeprecatedGauge *prometheus.GaugeVec
}

// NewMetrics function returns a pointer to a new Metrics struct that includes the initialized AvailableGauge and
// DeprecatedGauge.
func NewMetrics() *Metrics {
	return &Metrics{
		AvailableGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "aws_custom",
			Subsystem: "rds",
			Name:      "version_available",
			Help:      "Number of instances whose version is available",
		},
			[]string{"cluster_identifier", "engine", "engine_version"},
		),
		DeprecatedGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "aws_custom",
			Subsystem: "rds",
			Name:      "version_deprecated",
			Help:      "Number of instances whose Version is deprecated",
		},
			[]string{"cluster_identifier", "engine", "engine_version"},
		),
	}
}

// RDSInfo represents information about an Amazon RDS cluster.
type RDSInfo struct {
	// ClusterIdentifier is a unique identifier for the RDS cluster.
	ClusterIdentifier string

	// Engine is the name of the database engine used by the RDS cluster.
	// Examples of database engine names include "MySQL" and "PostgreSQL".
	Engine string

	// EngineVersion is the version of the database engine used by the RDS cluster.
	// Examples of database engine versions include "5.7.34" and "13.2".
	EngineVersion string
}

func main() {
	interval, err := getEnvInteger(AwsApiIntervalEnvName)
	if err != nil {
		log.Fatal(err)
	}

	port, err := getEnvInteger(ServerPortEnvName)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf(":%d", port)

	config := NewConfig()
	m, err := getEngineVersions(config)
	if err != nil {
		log.Fatal(err)
	}

	metrics := NewMetrics()
	handler := initPromHandler(metrics)
	server := initHttpServer(handler, addr)

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		// register metrics as background
		for range ticker.C {
			err := snapshot(config, metrics, m)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	log.Fatal(server.ListenAndServe())
}

// initPromHandler returns an HTTP handler that serves the Prometheus metrics defined in the Metrics struct. The handler
// uses the promhttp.Handler() function to generate an HTTP handler that serves the metrics in the correct format for
// Prometheus. The handler is wrapped with a logger to log requests to the metrics endpoint.
func initPromHandler(metrics *Metrics) http.Handler {
	r := prometheus.NewRegistry()
	r.MustRegister(metrics.AvailableGauge)
	r.MustRegister(metrics.DeprecatedGauge)
	return promhttp.HandlerFor(r, promhttp.HandlerOpts{})
}

// initHttpServer initializes the HTTP server that serves the Prometheus metrics. It sets up a new router, registers
// the Prometheus handler with the router, and then starts a new goroutine that listens for incoming HTTP requests
// on the specified port. If any error occurs during the setup process, the function will log the error and return it.
func initHttpServer(handler http.Handler, addr string) *http.Server {
	serveMux := http.NewServeMux()
	serveMux.Handle("/metrics", handler)
	return &http.Server{Addr: addr, Handler: serveMux}
}

// snapshot collects and exports metrics for all RDS instances and clusters.
// It first resets availableGauge and deprecatedGauge to zero, then fetches
// RDS cluster infos and RDS instance infos. It merges the infos into a single
// slice of RDSInfos, and exports the metrics for each RDSInfo. If any error
// occurs during the metric exporting process, the function will skip the
// problematic RDSInfo and continue exporting other RDSInfos.
//
// The function takes an argument of type engineVersions, which is a map
// containing a list of engine versions for each RDS engine type. It returns
// an error if any error occurs while reading the RDS cluster/instance info
// or while exporting the metrics.
func snapshot(config *Config, metrics *Metrics, m engineVersions) error {
	metrics.AvailableGauge.Reset()
	metrics.DeprecatedGauge.Reset()

	clusterInfos, err := getRDSClusters(config)
	if err != nil {
		return fmt.Errorf("failed to read RDS Cluster infos; %w", err)
	}

	InstanceInfos, err := getRDSInstances(config)
	if err != nil {
		return fmt.Errorf("failed to read RDS Instance infos; %w", err)
	}

	rdsInfos := clusterInfos
	rdsInfos = append(rdsInfos, InstanceInfos...)

	for _, rdsInfo := range rdsInfos {
		err := export(metrics, rdsInfo, m)
		if err != nil {
			return fmt.Errorf("skip: rdsInfo %#v; failed to export metric; %w", rdsInfo, err)
		}
	}

	return nil
}

// export collects RDS info and validates its engine version against the
// engineVersions struct that is provided. If the version is deprecated,
// it will set the deprecatedGauge prometheus metric to 1 and the availableGauge
// metric to 0. Otherwise, it sets the deprecatedGauge to 0 and the availableGauge
// to 1. It returns an error if the validation process or metric setting process fails.
//
// Example usage:
//
//	err := export(rdsInfo, engineVersions)
//	if err != nil {
//	    log.Printf("Failed to export RDS info: %v", err)
//	}
func export(metrics *Metrics, rdsInfo RDSInfo, m engineVersions) error {
	deprecated, err := validateEngineVersion(rdsInfo, m)
	if err != nil {
		return fmt.Errorf("failed to validate engine version: %w; skip rdsInfo: %#v", err, rdsInfo)
	}

	newLabels := prometheus.Labels{
		"cluster_identifier": rdsInfo.ClusterIdentifier,
		"engine":             rdsInfo.Engine,
		"engine_version":     rdsInfo.EngineVersion,
	}

	if deprecated {
		metrics.DeprecatedGauge.With(newLabels).Set(1)
		metrics.AvailableGauge.With(newLabels).Set(0)
	} else {
		metrics.DeprecatedGauge.With(newLabels).Set(0)
		metrics.AvailableGauge.With(newLabels).Set(1)
	}
	return nil
}

// getRDSClusters returns a slice of RDSInfo, which includes the identifiers and versions
// of all Amazon RDS clusters for the current AWS account and region.
// An error is returned if the function fails to retrieve cluster information.
func getRDSClusters(config *Config) ([]RDSInfo, error) {
	rdsInfos := make([]RDSInfo, 0)
	var nextMarker *string
	condition := true
	for condition {
		rdsClusters, err := config.RDS.DescribeDBClusters(&rds.DescribeDBClustersInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances; %w", err)
		}
		if rdsClusters == nil {
			break
		}
		rdsInfos = append(rdsInfos, handleRDSClusters(rdsClusters)...)
		nextMarker = rdsClusters.Marker
		condition = nextMarker != nil
	}
	return rdsInfos, nil
}

// handleRDSClusters receives a slice of RDSInfo structs representing Amazon RDS clusters and validates their engine
// version against a map of allowed engine versions. It updates the AvailableGauge and DeprecatedGauge Prometheus
// metrics accordingly. If an error occurs during the validation process, the function logs the error and continues
// processing other RDS clusters.
func handleRDSClusters(rdsClusters *rds.DescribeDBClustersOutput) []RDSInfo {
	rdsInfos := make([]RDSInfo, 0)
	for _, rdsCluster := range rdsClusters.DBClusters {
		RDSInfo := RDSInfo{
			ClusterIdentifier: *rdsCluster.DBClusterIdentifier,
			Engine:            *rdsCluster.Engine,
			EngineVersion:     *rdsCluster.EngineVersion,
		}
		rdsInfos = append(rdsInfos, RDSInfo)
	}
	return rdsInfos
}

// getRDSInstances retrieves information about all RDS instances in the AWS account
// and returns a slice of RDSInfo objects containing the ClusterIdentifier, Engine and EngineVersion.
// It uses the AWS SDK for Go to interact with the RDS service.
// If the function fails to retrieve the information, it returns an error.
func getRDSInstances(config *Config) ([]RDSInfo, error) {
	rdsInfos := make([]RDSInfo, 0)
	var nextMarker *string
	condition := true
	for condition {
		rdsInstances, err := config.RDS.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances; %w", err)
		}
		if rdsInstances == nil {
			break
		}
		rdsInfos = append(rdsInfos, handleRDSInstances(rdsInstances)...)
		nextMarker = rdsInstances.Marker
		condition = nextMarker != nil
	}
	return rdsInfos, nil
}

// handleRDSInstances receives a slice of RDSInfo structs representing Amazon RDS instances and validates their engine
// version against a map of allowed engine versions. It updates the AvailableGauge and DeprecatedGauge Prometheus
// metrics accordingly. If an error occurs during the validation process, the function logs the error and continues
// processing other RDS instances.
func handleRDSInstances(rdsInstances *rds.DescribeDBInstancesOutput) []RDSInfo {
	rdsInfos := make([]RDSInfo, 0)
	for _, rdsInstance := range rdsInstances.DBInstances {
		RDSInfo := RDSInfo{
			ClusterIdentifier: *rdsInstance.DBInstanceIdentifier,
			Engine:            *rdsInstance.Engine,
			EngineVersion:     *rdsInstance.EngineVersion,
		}
		rdsInfos = append(rdsInfos, RDSInfo)
	}
	return rdsInfos
}
