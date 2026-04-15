/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"sync"

	"helm.sh/helm/v3/pkg/cli"
)

var (
	settings     *cli.EnvSettings
	settingsOnce sync.Once
)

// helmSettings returns a shared Helm CLI EnvSettings instance.
func helmSettings() *cli.EnvSettings {
	settingsOnce.Do(func() {
		settings = cli.New()
	})
	return settings
}