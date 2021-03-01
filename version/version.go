// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package version

// Version describes the version of Shipwright build controller
var Version = ""

// SetVersion sets the version of Shipwright build controller from go flags
func SetVersion(version string) {
	Version = version
}
