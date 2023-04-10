package main

import (
	"fmt"
	"os"
	"strconv"
)

func Ptr[T any](v T) *T {
	return &v
}

// getEnvInteger retrieves the value of an environment variable with the given name and returns it as an integer.
// If the variable is not set, or if its value cannot be parsed as an integer, an error will be returned.
func getEnvInteger(name string) (int, error) {
	interval := os.Getenv(name)
	if len(interval) == 0 {
		return 0, fmt.Errorf("environment variable %s should be set", name)
	}

	parsedInterval, err := strconv.Atoi(interval)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s could not be parsed: %w", name, err)
	}
	return parsedInterval, nil
}
