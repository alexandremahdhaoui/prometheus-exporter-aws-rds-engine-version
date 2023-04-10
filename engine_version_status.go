package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/rds"
)

// versionDeprecations is mapping RDS engine versions to their deprecation status. A version will be mapped to true if
// it's deprecated.
type versionDeprecations map[string]bool

// engineVersions is mapping an RDS Engine to its available versionDeprecations
type engineVersions map[string]versionDeprecations

// getEngineVersions() returns a map of RDS engine versions and their deprecation status, represented by a nested
// map of engineVersions and versionDeprecations.
//
// The engineVersions is a map of RDS engine names to versionDeprecations, which is another map of RDS engine versions
// to boolean values representing whether that version is deprecated or not.
//
// The function populates this map by calling queryEngineVersions() twice with false as the first parameter,
// passing in the engineVersions map as the second parameter. If an error occurs during either of the calls to
// queryEngineVersions(), an error is returned.
func getEngineVersions(config *Config) (engineVersions, error) {
	m := make(engineVersions)

	if err := queryEngineVersions(config, false, m); err != nil {
		return nil, fmt.Errorf("error while querying rds engine version status; %w", err)
	}
	if err := queryEngineVersions(config, false, m); err != nil {
		return nil, fmt.Errorf("error while querying rds engine version status; %w", err)
	}

	return m, nil
}

// queryEngineVersions() queries the AWS RDS API to get information about the deprecation status of engine
// versions, as determined by the deprecatedVersion boolean parameter.
//
// The function takes in a map of engineVersions as a second parameter, which is used to store the deprecation status
// of each RDS engine version.
//
// The function creates an AWS session and RDS client using the AWS SDK for Go. It then loops over all pages of the RDS
// engine versions using the DescribeDBEngineVersions API method with a filter on the status field set to either
// "available" or "deprecated", depending on the deprecatedVersion parameter.
//
// For each RDS engine version, the function updates the engineVersions map with the deprecation status of that version.
// If the RDS engine is not already in the map, it creates a new versionDeprecations map to store the deprecation
// status of that engine's versions.
//
// If any error occurs while querying the RDS API or updating the engineVersions map, an error is returned.
//
// Overall, this function is responsible for populating the engineVersions map with deprecation status information
// retrieved from the AWS RDS API.
func queryEngineVersions(config *Config, deprecatedVersion bool, m engineVersions) error {
	status := evalStatus(deprecatedVersion)

	var nextMarker *string
	cond := true
	for cond {
		dbEngineVersions, err := config.RDS.DescribeDBEngineVersions(&rds.DescribeDBEngineVersionsInput{
			Filters: []*rds.Filter{{
				Name:   Ptr("status"),
				Values: []*string{&status},
			}},
			Marker: nextMarker,
		})
		if err != nil {
			return fmt.Errorf("failed to describe db engine versions; %w", err)
		}
		if dbEngineVersions == nil {
			break
		}
		for _, dbEngineVersion := range dbEngineVersions.DBEngineVersions {
			if deprecationMap, ok := m[*dbEngineVersion.Engine]; ok {
				deprecationMap[*dbEngineVersion.EngineVersion] = deprecatedVersion
			} else {
				deprecationMap := make(versionDeprecations)
				deprecationMap[*dbEngineVersion.EngineVersion] = deprecatedVersion
				m[*dbEngineVersion.Engine] = deprecationMap
			}
		}
		nextMarker = dbEngineVersions.Marker
		cond = nextMarker != nil
	}
	return nil
}

func evalStatus(deprecated bool) string {
	if deprecated {
		return "deprecated"
	} else {
		return "available"
	}
}

// ---------------------------------------------------------------------------------------------------------------------

// validateEngineVersion() takes in an RDSInfo struct that contains information about an RDS engine, and an
// engineVersions map that contains deprecation status information for all RDS engines and versions.
//
// The function first checks if the RDS engine in the RDSInfo struct is present in the engineVersions map. If it is not,
// the function returns false and an error indicating that the engine is unknown.
//
// If the engine is present in the engineVersions map, the function then checks if the version of the RDS engine in the
// RDSInfo struct is present in the versionDeprecations map for that engine. If it is not, the function returns false
// and an error indicating that the version is unknown.
//
// If the engine and version are present in the engineVersions map, the function returns a boolean indicating whether
// the version is deprecated or not, based on the deprecation status value stored in the versionDeprecations map.
//
// Overall, this function is responsible for validating an RDS engine and version by checking if they are present in the
// engineVersions map and returning whether the version is deprecated or not.
func validateEngineVersion(rdsInfo RDSInfo, m engineVersions) (bool, error) {
	if _, ok := m[rdsInfo.Engine]; !ok {
		return false, fmt.Errorf("unknown engine: %s; failed to validate RDS Engine version", rdsInfo.Engine)
	}
	versions := m[rdsInfo.Engine]

	if _, ok := versions[rdsInfo.EngineVersion]; !ok {
		return false, fmt.Errorf("unknown version: %s; failed to validate RDS Engine version", rdsInfo.EngineVersion)
	}
	return !versions[rdsInfo.EngineVersion], nil
}
