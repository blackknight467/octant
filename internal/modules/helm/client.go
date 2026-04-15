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
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/vmware-tanzu/octant/pkg/config"
)

// helmClient wraps the helm action configuration and provides release operations.
type helmClient struct {
	cfg *helmAction.Configuration
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

	return &helmClient{cfg: cfg}, nil
}

// listReleases returns all Helm releases across all namespaces.
func (c *helmClient) listReleases() ([]*release.Release, error) {
	list := helmAction.NewList(c.cfg)
	list.AllNamespaces = true
	list.All = true
	list.SetStateMask()
	return list.Run()
}

// getRelease returns a specific release by name.
func (c *helmClient) getRelease(name string) (*release.Release, error) {
	get := helmAction.NewGet(c.cfg)
	return get.Run(name)
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