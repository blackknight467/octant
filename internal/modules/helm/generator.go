/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/octant/internal/describer"
	"github.com/vmware-tanzu/octant/pkg/config"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// helmGenerator generates content for helm module paths.
type helmGenerator struct {
	pathMatcher *describer.PathMatcher
	dashConfig  config.Dash
}

func newHelmGenerator(pm *describer.PathMatcher, dashConfig config.Dash) (*helmGenerator, error) {
	return &helmGenerator{pathMatcher: pm, dashConfig: dashConfig}, nil
}

// Generate produces a ContentResponse for the given contentPath.
func (g *helmGenerator) Generate(ctx context.Context, contentPath string) (component.ContentResponse, error) {
	// Content paths arrive as "helm/..." or "/helm/..." — strip the module prefix either way.
	path := strings.TrimPrefix(contentPath, "/"+moduleName)
	path = strings.TrimPrefix(path, moduleName)
	if path == "" || path == "/" {
		path = "/"
	} else if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	pf, err := g.pathMatcher.Find(path)
	if err != nil {
		return component.EmptyContentResponse, errors.Wrapf(err, "find path %q", path)
	}

	options := describer.Options{
		Fields: pf.Fields(path),
	}

	cResponse, err := pf.Describer.Describe(ctx, "", options)
	if err != nil {
		return component.EmptyContentResponse, errors.Wrapf(err, "describe path %q", path)
	}

	return cResponse, nil
}
