/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"
	"strings"

	"github.com/vmware-tanzu/octant/internal/describer"
	"github.com/vmware-tanzu/octant/pkg/config"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// releaseDescriber describes a single Helm release detail view.
type releaseDescriber struct {
	dashConfig config.Dash
}

var _ describer.Describer = (*releaseDescriber)(nil)

func newReleaseDescriber(dashConfig config.Dash) *releaseDescriber {
	return &releaseDescriber{dashConfig: dashConfig}
}

// Describe builds the release detail view.
func (d *releaseDescriber) Describe(ctx context.Context, namespace string, options describer.Options) (component.ContentResponse, error) {
	// Extract release name and namespace from path: /helm/<ns>/<name>
	relNamespace, relName, err := parseReleasePath(options.Fields)
	if err != nil {
		return component.EmptyContentResponse, err
	}

	client, err := newHelmClient(d.dashConfig, relNamespace)
	if err != nil {
		return component.EmptyContentResponse, fmt.Errorf("helm client: %w", err)
	}

	rel, err := client.getRelease(relName)
	if err != nil {
		return component.EmptyContentResponse, fmt.Errorf("get release %q: %w", relName, err)
	}

	// Summary card
	var sections []component.SummarySection
	if rel.Info != nil {
		sections = append(sections,
			component.SummarySection{Header: "Status", Content: component.NewText(string(rel.Info.Status))},
			component.SummarySection{Header: "Last Deployed", Content: component.NewText(rel.Info.LastDeployed.String())},
			component.SummarySection{Header: "Description", Content: component.NewText(rel.Info.Description)},
		)
	}
	sections = append(sections,
		component.SummarySection{Header: "Namespace", Content: component.NewText(rel.Namespace)},
		component.SummarySection{Header: "Revision", Content: component.NewText(fmt.Sprintf("%d", rel.Version))},
	)
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		sections = append(sections,
			component.SummarySection{Header: "Chart", Content: component.NewText(fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version))},
			component.SummarySection{Header: "App Version", Content: component.NewText(rel.Chart.Metadata.AppVersion)},
		)
	}
	summary := component.NewSummary(fmt.Sprintf("Release: %s", relName), sections...)

	var components []component.Component
	components = append(components, summary)

	// Release notes
	if rel.Info != nil && rel.Info.Notes != "" {
		notesText := component.NewText(rel.Info.Notes)
		notesCard := component.NewCard(component.TitleFromString("Release Notes"))
		notesCard.SetBody(notesText)
		components = append(components, notesCard)
	}

	// History table
	history, err := client.getReleaseHistory(relName)
	if err == nil && len(history) > 0 {
		cols := component.NewTableCols("Revision", "Updated", "Status", "Chart", "App Version", "Description")
		histTable := component.NewTable("Revision History", "No history available", cols)
		for _, h := range history {
			if h.Info == nil {
				continue
			}
			chartName := ""
			appVersion := ""
			if h.Chart != nil && h.Chart.Metadata != nil {
				chartName = fmt.Sprintf("%s-%s", h.Chart.Metadata.Name, h.Chart.Metadata.Version)
				appVersion = h.Chart.Metadata.AppVersion
			}
			histTable.Add(component.TableRow{
				"Revision":    component.NewText(fmt.Sprintf("%d", h.Version)),
				"Updated":     component.NewText(h.Info.LastDeployed.String()),
				"Status":      component.NewText(string(h.Info.Status)),
				"Chart":       component.NewText(chartName),
				"App Version": component.NewText(appVersion),
				"Description": component.NewText(h.Info.Description),
			})
		}
		components = append(components, histTable)
	}

	return component.ContentResponse{
		Title:      component.TitleFromString(fmt.Sprintf("Helm Release: %s", relName)),
		Components: components,
	}, nil
}

// PathFilters returns path filters for individual release detail pages.
func (d *releaseDescriber) PathFilters() []describer.PathFilter {
	return []describer.PathFilter{
		*describer.NewPathFilter("/:namespace/:name", d),
	}
}

// Reset does nothing.
func (d *releaseDescriber) Reset(ctx context.Context) error {
	return nil
}

func parseReleasePath(fields map[string]string) (namespace, name string, err error) {
	name = fields["name"]
	namespace = fields["namespace"]
	if name == "" || namespace == "" {
		return "", "", fmt.Errorf("missing release name or namespace in path (fields: %v)", fields)
	}
	name = strings.TrimPrefix(name, "/")
	namespace = strings.TrimPrefix(namespace, "/")
	return namespace, name, nil
}