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
					"1.0": false,
				},
				"engine2": {
					"2.0": false,
				},
				"engine3": {
					"3.0": false,
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
			wantErr: errors.New("error while querying rds engine version status; failed to describe db engine versions; failed to describe db engine versions"),
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
