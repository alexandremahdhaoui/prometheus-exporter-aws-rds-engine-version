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
	"errors"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestValidateEngineVersion tests the validateEngineVersion function.
func TestValidateEngineVersion(t *testing.T) {
	// define test cases
	tests := []struct {
		name    string
		rdsInfo RDSInfo
		m       engineVersions
		want    bool
		wantErr bool
	}{
		{
			name: "valid engine and version; deprecated",
			rdsInfo: RDSInfo{
				Engine:        "mysql",
				EngineVersion: "5.1.1",
			},
			m: engineVersions{
				"mysql": versionDeprecations{
					"5.1.1": true,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "valid engine and version; not deprecated",
			rdsInfo: RDSInfo{
				Engine:        "mysql",
				EngineVersion: "5.5.5",
			},
			m: engineVersions{
				"mysql": versionDeprecations{
					"5.5.5": false,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unknown engine",
			rdsInfo: RDSInfo{
				Engine:        "foo",
				EngineVersion: "1.0",
			},
			m:       engineVersions{},
			want:    false,
			wantErr: true,
		},
		{
			name: "unknown engine version",
			rdsInfo: RDSInfo{
				Engine:        "mysql",
				EngineVersion: "foo",
			},
			m: engineVersions{
				"mysql": versionDeprecations{
					"5.5.5": false,
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	// run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateEngineVersion(tt.rdsInfo, tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEngineVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateEngineVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//--------------------------------------------------------------------------------------------------------------------
//--------------------------------------------------------------------------------------------------------------------

// TestGetEngineVersions tests the getEngineVersions function.
func TestGetEngineVersions(t *testing.T) {
	tests := []struct {
		desc    string
		config  *Config
		want    engineVersions
		wantErr error
	}{
		{
			desc: "successful query",
			config: &Config{
				RDS: &MockRDSAPI{
					engineVersionsOutput: []*rds.DescribeDBEngineVersionsOutput{
						{
							DBEngineVersions: []*rds.DBEngineVersion{
								{
									Engine:        Ptr("engine1"),
									EngineVersion: Ptr("1.0"),
								},
								{
									Engine:        Ptr("engine2"),
									EngineVersion: Ptr("2.0"),
								},
							},
							Marker: Ptr("yolo"),
						},
						{
							DBEngineVersions: []*rds.DBEngineVersion{
								{
									Engine:        Ptr("engine3"),
									EngineVersion: Ptr("3.0"),
								},
							},
							Marker: nil,
						},
					},
				},
			},
			want: engineVersions{
				"engine1": {
					"1.0": true,
				},
				"engine2": {
					"2.0": true,
				},
				"engine3": {
					"3.0": true,
				},
			},
			wantErr: nil,
		},
		{
			desc: "failed query",
			config: &Config{
				RDS: &MockRDSAPI{
					err: errors.New("failed to describe db engine versions"),
				},
			},
			want:    nil,
			wantErr: errors.New("error while querying rds deprecated engine version; failed to describe db engine versions; failed to describe db engine versions"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Logf("testing: %s", tt.desc)

			got, err := getEngineVersions(tt.config)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
