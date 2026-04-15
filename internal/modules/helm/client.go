/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"fmt"
	"os"

	helmAction "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/vmware-tanzu/octant/pkg/config"
)

// helmClient wraps the helm action configuration and provides release operations.
type helmClient struct {
	cfg        *helmAction.Configuration
	dashConfig config.Dash
	namespace  string
}

// newHelmClient creates a helm action configuration using the kubeconfig from dashConfig.
func newHelmClient(dashConfig config.Dash, namespace string) (*helmClient, error) {
	kubeConfigPath := dashConfig.KubeConfigPath()

	configFlags := &genericclioptions.ConfigFlags{
		KubeConfig: &kubeConfigPath,
		Namespace:  &namespace,
	}

	cfg := &helmAction.Configuration{}
	if err := cfg.Init(configFlags, namespace, "secret", func(format string, v ...interface{}) {
		fmt.Fprintf(os.Stderr, format+"\n", v...)
	}); err != nil {
		return nil, err
	}

	return &helmClient{cfg: cfg, dashConfig: dashConfig, namespace: namespace}, nil
}

// listReleases returns all Helm releases across all namespaces.
func (c *helmClient) listReleases() ([]*release.Release, error) {
	list := helmAction.NewList(c.cfg)
	list.AllNamespaces = true
	list.All = true
	list.SetStateMask()
	return list.Run()
}

// listReleasesInNamespace returns Helm releases for a specific namespace.
func (c *helmClient) listReleasesInNamespace(namespace string) ([]*release.Release, error) {
	list := helmAction.NewList(c.cfg)
	list.All = true
	list.SetStateMask()
	releases, err := list.Run()
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		return releases, nil
	}
	var filtered []*release.Release
	for _, r := range releases {
		if r.Namespace == namespace {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// getRelease returns a specific release by name.
func (c *helmClient) getRelease(name string) (*release.Release, error) {
	get := helmAction.NewGet(c.cfg)
	return get.Run(name)
}

// getReleaseValues returns the user-supplied values for a release.
func (c *helmClient) getReleaseValues(name string, allValues bool) (map[string]interface{}, error) {
	getValues := helmAction.NewGetValues(c.cfg)
	getValues.AllValues = allValues
	return getValues.Run(name)
}

// getReleaseHistory returns all revisions for a named release.
func (c *helmClient) getReleaseHistory(name string) ([]*release.Release, error) {
	hist := helmAction.NewHistory(c.cfg)
	hist.Max = 256
	return hist.Run(name)
}

// uninstallRelease removes a named Helm release.
func (c *helmClient) uninstallRelease(name string) (*release.UninstallReleaseResponse, error) {
	uninstall := helmAction.NewUninstall(c.cfg)
	return uninstall.Run(name)
}

// rollbackRelease rolls back a release to a specific version.
func (c *helmClient) rollbackRelease(name string, version int) error {
	rollback := helmAction.NewRollback(c.cfg)
	rollback.Version = version
	return rollback.Run(name)
}

// upgradeRelease upgrades a release with new values.
func (c *helmClient) upgradeRelease(name string, values map[string]interface{}) (*release.Release, error) {
	// Get the current release to obtain the chart
	current, err := c.getRelease(name)
	if err != nil {
		return nil, fmt.Errorf("get current release: %w", err)
	}
	upgrade := helmAction.NewUpgrade(c.cfg)
	upgrade.ReuseValues = true
	return upgrade.Run(name, current.Chart, values)
}

// runTests runs the helm tests for a release.
func (c *helmClient) runTests(name string) (*release.Release, error) {
	test := helmAction.NewReleaseTesting(c.cfg)
	return test.Run(name)
}

// listRepositories reads the local Helm repositories file.
func listRepositories(dashConfig config.Dash) ([]*repo.Entry, error) {
	path := helmRepositoriesPath()
	repoFile, err := repo.LoadFile(path)
	if err != nil {
		// Not an error if the file doesn't exist yet
		return nil, nil
	}
	return repoFile.Repositories, nil
}

// removeRepository removes a repository entry by name from the repositories file.
func removeRepository(name string) error {
	path := helmRepositoriesPath()
	repoFile, err := repo.LoadFile(path)
	if err != nil {
		return err
	}
	repoFile.Remove(name)
	return repoFile.WriteFile(path, 0o600)
}

// helmRepositoriesPath returns the path to the Helm repositories.yaml file.
func helmRepositoriesPath() string {
	settings := helmSettings()
	return settings.RepositoryConfig
}