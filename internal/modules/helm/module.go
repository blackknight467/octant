/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/octant/internal/describer"
	"github.com/vmware-tanzu/octant/internal/module"
	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/config"
	"github.com/vmware-tanzu/octant/pkg/navigation"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

const (
	moduleName = "helm"
)

// Options are options for configuring the Helm Module.
type Options struct {
	DashConfig config.Dash
}

// Module is the native Helm integration module.
type Module struct {
	Options
	namespace   string
	pathMatcher *describer.PathMatcher
}

var _ module.Module = (*Module)(nil)
var _ module.ActionReceiver = (*Module)(nil)

// New creates an instance of the Helm Module.
func New(ctx context.Context, options Options) (*Module, error) {
	m := &Module{
		Options: options,
	}

	pm := describer.NewPathMatcher(moduleName)
	releasesDescriber := newReleasesDescriber(options.DashConfig)
	for _, pf := range releasesDescriber.PathFilters() {
		pm.Register(ctx, pf)
	}
	releaseDescriber := newReleaseDescriber(options.DashConfig)
	for _, pf := range releaseDescriber.PathFilters() {
		pm.Register(ctx, pf)
	}

	m.pathMatcher = pm
	return m, nil
}

// Name returns the name of the module.
func (m *Module) Name() string {
	return moduleName
}

// Description returns the description of the module.
func (m *Module) Description() string {
	return "Native Helm release management"
}

// ClientRequestHandlers returns client request handlers.
func (m *Module) ClientRequestHandlers() []octant.ClientRequestHandler {
	return nil
}

// ContentPath returns the root content path.
func (m *Module) ContentPath() string {
	return moduleName
}

// Content generates content for a given path.
func (m *Module) Content(ctx context.Context, contentPath string, opts module.ContentOptions) (component.ContentResponse, error) {
	g, err := newHelmGenerator(m.pathMatcher, m.DashConfig)
	if err != nil {
		return component.EmptyContentResponse, err
	}
	return g.Generate(ctx, contentPath)
}

// Navigation generates navigation entries.
func (m *Module) Navigation(ctx context.Context, namespace, root string) ([]navigation.Navigation, error) {
	rootPath := fmt.Sprintf("/%s", moduleName)
	nav := navigation.Navigation{
		Title:    "Helm Releases",
		Path:     rootPath,
		IconName: "application",
	}
	return []navigation.Navigation{nav}, nil
}

// SetNamespace stores the current namespace.
func (m *Module) SetNamespace(namespace string) error {
	m.namespace = namespace
	return nil
}

// Start does nothing.
func (m *Module) Start() error {
	return nil
}

// Stop does nothing.
func (m *Module) Stop() {}

// SetContext does nothing.
func (m *Module) SetContext(ctx context.Context, contextName string) error {
	return nil
}

// Generators does nothing.
func (m *Module) Generators() []octant.Generator {
	return nil
}

// SupportedGroupVersionKind returns nil (Helm releases are not k8s objects).
func (m *Module) SupportedGroupVersionKind() []schema.GroupVersionKind {
	return nil
}

// GroupVersionKindPath returns an error (not supported).
func (m *Module) GroupVersionKindPath(namespace, apiVersion, kind, name string) (string, error) {
	return "", errors.New("not supported")
}

// AddCRD does nothing.
func (m *Module) AddCRD(ctx context.Context, crd *unstructured.Unstructured) error {
	return nil
}

// RemoveCRD does nothing.
func (m *Module) RemoveCRD(ctx context.Context, crd *unstructured.Unstructured) error {
	return nil
}

// ResetCRDs does nothing.
func (m *Module) ResetCRDs(ctx context.Context) error {
	return nil
}

// GvkFromPath returns an error (not supported).
func (m *Module) GvkFromPath(contentPath, namespace string) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, errors.New("not supported")
}

// ActionPaths returns action handlers for Helm operations.
func (m *Module) ActionPaths() map[string]action.DispatcherFunc {
	dispatchers := action.Dispatchers{
		newUninstallAction(m.DashConfig),
	}
	return dispatchers.ToActionPaths()
}