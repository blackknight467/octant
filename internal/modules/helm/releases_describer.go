/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"

	"github.com/vmware-tanzu/octant/internal/describer"
	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/config"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// releasesDescriber describes all Helm releases.
type releasesDescriber struct {
	dashConfig config.Dash
}

var _ describer.Describer = (*releasesDescriber)(nil)

func newReleasesDescriber(dashConfig config.Dash) *releasesDescriber {
	return &releasesDescriber{dashConfig: dashConfig}
}

// Describe builds the releases list table.
func (d *releasesDescriber) Describe(ctx context.Context, namespace string, options describer.Options) (component.ContentResponse, error) {
	client, err := newHelmClient(d.dashConfig, "")
	if err != nil {
		return component.EmptyContentResponse, fmt.Errorf("helm client: %w", err)
	}

	releases, err := client.listReleases()
	if err != nil {
		return component.EmptyContentResponse, fmt.Errorf("list releases: %w", err)
	}

	cols := component.NewTableCols("Name", "Namespace", "Revision", "Updated", "Status", "Chart", "App Version", "Actions")
	table := component.NewTable("Helm Releases", "No Helm releases found", cols)

	for _, rel := range releases {
		if rel.Info == nil {
			continue
		}

		detailPath := fmt.Sprintf("/helm/%s/%s", rel.Namespace, rel.Name)
		uninstallPayload := action.Payload{
			"action":    ActionHelmUninstall,
			"name":      rel.Name,
			"namespace": rel.Namespace,
		}
		uninstallBtn := component.NewButton("Uninstall", uninstallPayload,
			component.WithButtonConfirmation(
				"Uninstall Release",
				fmt.Sprintf("Are you sure you want to uninstall release %q?", rel.Name),
			),
		)
		btnGroup := component.NewButtonGroup()
		btnGroup.AddButton(uninstallBtn)

		chartName := ""
		appVersion := ""
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			chartName = fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)
			appVersion = rel.Chart.Metadata.AppVersion
		}

		table.Add(component.TableRow{
			"Name":        component.NewLink("", rel.Name, detailPath),
			"Namespace":   component.NewText(rel.Namespace),
			"Revision":    component.NewText(fmt.Sprintf("%d", rel.Version)),
			"Updated":     component.NewText(rel.Info.LastDeployed.String()),
			"Status":      component.NewText(string(rel.Info.Status)),
			"Chart":       component.NewText(chartName),
			"App Version": component.NewText(appVersion),
			"Actions":     btnGroup,
		})
	}

	return component.ContentResponse{
		Title:      component.TitleFromString("Helm Releases"),
		Components: []component.Component{table},
	}, nil
}

// PathFilters returns path filters for the root releases list.
func (d *releasesDescriber) PathFilters() []describer.PathFilter {
	return []describer.PathFilter{
		*describer.NewPathFilter("/", d),
	}
}

// Reset does nothing.
func (d *releasesDescriber) Reset(ctx context.Context) error {
	return nil
}