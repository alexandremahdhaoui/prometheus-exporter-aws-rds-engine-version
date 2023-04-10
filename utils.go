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
