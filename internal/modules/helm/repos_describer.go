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

// reposDescriber describes the Helm repositories view.
type reposDescriber struct {
	dashConfig config.Dash
}

var _ describer.Describer = (*reposDescriber)(nil)

func newReposDescriber(dashConfig config.Dash) *reposDescriber {
	return &reposDescriber{dashConfig: dashConfig}
}

// Describe builds the repositories list table.
func (d *reposDescriber) Describe(ctx context.Context, namespace string, options describer.Options) (component.ContentResponse, error) {
	repos, err := listRepositories(d.dashConfig)
	if err != nil {
		return component.EmptyContentResponse, fmt.Errorf("list repositories: %w", err)
	}

	cols := component.NewTableCols("Name", "URL", "Actions")
	table := component.NewTable("Helm Repositories", "No Helm repositories configured. Use `helm repo add` to add repositories.", cols)

	for _, repo := range repos {
		removePayload := action.Payload{
			"action": ActionHelmRemoveRepo,
			"name":   repo.Name,
		}
		removeBtn := component.NewButton("Remove", removePayload,
			component.WithButtonConfirmation(
				"Remove Repository",
				fmt.Sprintf("Remove repository %q?", repo.Name),
			),
		)
		btnGroup := component.NewButtonGroup()
		btnGroup.AddButton(removeBtn)

		table.Add(component.TableRow{
			"Name":    component.NewText(repo.Name),
			"URL":     component.NewText(repo.URL),
			"Actions": btnGroup,
		})
	}

	return component.ContentResponse{
		Title:      component.TitleFromString("Helm Repositories"),
		Components: []component.Component{table},
	}, nil
}

// PathFilters returns path filters for the repos page.
func (d *reposDescriber) PathFilters() []describer.PathFilter {
	return []describer.PathFilter{
		*describer.NewPathFilter("/repos", d),
	}
}

// Reset does nothing.
func (d *reposDescriber) Reset(ctx context.Context) error { return nil }