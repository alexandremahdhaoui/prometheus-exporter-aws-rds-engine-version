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

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"os"
	"testing"
)

const serverPort = "2112"
const awsApiInterval = "1"
const metricsPath = "/metrics"

// Mocks

type MockRDSAPI struct {
	rdsiface.RDSAPI
	instancesOutput      []*rds.DescribeDBInstancesOutput
	clustersOutput       []*rds.DescribeDBClustersOutput
	engineVersionsOutput []*rds.DescribeDBEngineVersionsOutput
	err                  error
}

func (m MockRDSAPI) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	return getSafe(m.instancesOutput, input.Marker, m.err)
}
func (m MockRDSAPI) DescribeDBClusters(input *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
	return getSafe(m.clustersOutput, input.Marker, m.err)
}

func (m MockRDSAPI) DescribeDBEngineVersions(input *rds.DescribeDBEngineVersionsInput) (*rds.DescribeDBEngineVersionsOutput, error) {
	return getSafe(m.engineVersionsOutput, input.Marker, m.err)
}

func getSafe[T []*Y, Y any](v T, inputMarker *string, err error) (*Y, error) {
	if err != nil {
		return nil, err
	}
	// Marker is nil when calling the first time the RDS API
	if inputMarker == nil {
		if len(v) < 1 {
			return nil, nil
		}
		return v[0], nil

	}
	if len(v) < 2 {
		return nil, nil
	}
	return v[1], nil
}

// Tests

func TestMain(m *testing.M) {
	t := &testing.T{}
	setEnv(t, AwsApiIntervalEnvName, awsApiInterval)
	setEnv(t, ServerPortEnvName, serverPort)
	code := m.Run()
	os.Exit(code)
}

func TestGetEnvInteger(t *testing.T) {
	// Test with valid integer string
	setEnv(t, "TEST_VAR", "123")
	i, err := getEnvInteger("TEST_VAR")
	assert.NoError(t, err)
	assert.Equal(t, 123, i)

	// Test with invalid integer string
	setEnv(t, "TEST_VAR", "foo")
	_, err = getEnvInteger("TEST_VAR")
	assert.Error(t, err)
}

func TestSnapshot(t *testing.T) {
	m := engineVersions{
		"MySQL":      {"5.7.34": true, "8.0.25": false},
		"PostgreSQL": {"9.5.24": true, "13.2": false},
	}
	tests := []struct {
		desc    string
		config  *Config
		want    string
		wantErr error
	}{
		{
			desc: "successful snapshot",
			config: &Config{RDS: &MockRDSAPI{
				instancesOutput: []*rds.DescribeDBInstancesOutput{
					{
						DBInstances: []*rds.DBInstance{
							{
								DBInstanceIdentifier: Ptr("cluster-1"),
								Engine:               Ptr("MySQL"),
								EngineVersion:        Ptr("5.7.34"),
							},
							{
								DBInstanceIdentifier: Ptr("cluster-1"),
								Engine:               Ptr("MySQL"),
								EngineVersion:        Ptr("8.0.25"),
							},
						},
						Marker: Ptr("dummy marker"),
					},
					{
						DBInstances: []*rds.DBInstance{
							{
								DBInstanceIdentifier: Ptr("cluster-1"),
								Engine:               Ptr("PostgreSQL"),
								EngineVersion:        Ptr("9.5.24"),
							},
							{
								DBInstanceIdentifier: Ptr("cluster-1"),
								Engine:               Ptr("PostgreSQL"),
								EngineVersion:        Ptr("13.2"),
							},
						},
						Marker: nil,
					},
				},
			}},
			want: `# HELP aws_custom_rds_version_available Number of instances whose version is available
# TYPE aws_custom_rds_version_available gauge
aws_custom_rds_version_available{cluster_identifier="cluster-1",engine="MySQL",engine_version="5.7.34"} 0
aws_custom_rds_version_available{cluster_identifier="cluster-1",engine="MySQL",engine_version="8.0.25"} 1
aws_custom_rds_version_available{cluster_identifier="cluster-1",engine="PostgreSQL",engine_version="13.2"} 1
aws_custom_rds_version_available{cluster_identifier="cluster-1",engine="PostgreSQL",engine_version="9.5.24"} 0
# HELP aws_custom_rds_version_deprecated Number of instances whose Version is deprecated
# TYPE aws_custom_rds_version_deprecated gauge
aws_custom_rds_version_deprecated{cluster_identifier="cluster-1",engine="MySQL",engine_version="5.7.34"} 1
aws_custom_rds_version_deprecated{cluster_identifier="cluster-1",engine="MySQL",engine_version="8.0.25"} 0
aws_custom_rds_version_deprecated{cluster_identifier="cluster-1",engine="PostgreSQL",engine_version="13.2"} 0
aws_custom_rds_version_deprecated{cluster_identifier="cluster-1",engine="PostgreSQL",engine_version="9.5.24"} 1
`,
			wantErr: nil,
		},
		{
			desc:    "failed snapshot getRDSClusters returns error",
			config:  &Config{&MockRDSAPI{err: fmt.Errorf("failed to get clusters")}},
			want:    "",
			wantErr: errors.New("failed to read RDS Cluster infos; failed to describe DB instances; failed to get clusters"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Logf("testing: %s", tt.desc)

			metrics := NewMetrics()
			handler := initPromHandler(metrics)
			server := initHttpServer(handler, getAddr())
			go func() {
				_ = server.ListenAndServe()
			}()

			err := snapshot(tt.config, metrics, m)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			got := queryPrometheusServer(t)
			assert.Equal(t, tt.want, got)
			err = server.Shutdown(context.TODO())
			assert.NoError(t, err)
		})
	}
}

func setEnv(t *testing.T, key, value string) {
	err := os.Setenv(key, value)
	assert.NoError(t, err)
}

func queryPrometheusServer(t *testing.T) string {
	get, err := http.Get(getMetricsUrl())
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(get.Body)
	return string(b)
}

func getAddr() string {
	return fmt.Sprintf(":%s", serverPort)
}

func getMetricsUrl() string {
	return fmt.Sprintf("http://127.0.0.1%s%s", getAddr(), metricsPath)
}
