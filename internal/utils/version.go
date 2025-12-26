/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// CompareVersions compares two semantic version strings
// Returns:
// - negative number if v1 < v2
// - zero if v1 == v2
// - positive number if v1 > v2
func CompareVersions(v1, v2 string) (int, error) {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		var err error

		if i < len(parts1) {
			n1, err = strconv.Atoi(parts1[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version format: %s", v1)
			}
		}

		if i < len(parts2) {
			n2, err = strconv.Atoi(parts2[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version format: %s", v2)
			}
		}

		if n1 < n2 {
			return -1, nil
		}
		if n1 > n2 {
			return 1, nil
		}
	}

	return 0, nil
}

// IsVersionDowngrade checks if newVersion is a downgrade from currentVersion
func IsVersionDowngrade(currentVersion, newVersion string) (bool, error) {
	cmp, err := CompareVersions(currentVersion, newVersion)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
