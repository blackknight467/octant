/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	helmRelease "helm.sh/helm/v3/pkg/release"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/vmware-tanzu/octant/internal/describer"
	"github.com/vmware-tanzu/octant/pkg/action"
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

	var components []component.Component

	// Failed release diagnostic banner
	if rel.Info != nil && rel.Info.Status == helmRelease.StatusFailed {
		errCard := component.NewCard(component.TitleFromString("Release Failed"))
		errCard.SetBody(component.NewText(fmt.Sprintf("Status: %s\n\n%s", rel.Info.Status, rel.Info.Description)))
		components = append(components, errCard)
	}

	// Summary card
	components = append(components, buildSummaryCard(rel, relName, relNamespace))

	// Action buttons (test + upgrade notice)
	testPayload := action.Payload{
		"action":    ActionHelmTest,
		"name":      relName,
		"namespace": relNamespace,
	}
	testBtn := component.NewButton("Run Tests", testPayload)
	btnGroup := component.NewButtonGroup()
	btnGroup.AddButton(testBtn)
	components = append(components, btnGroup)

	// Values card (computed: user + defaults merged)
	components = append(components, buildValuesCard(client, relName, relNamespace))

	// Upgrade form
	components = append(components, buildUpgradeCard(client, relName, relNamespace))

	// Release notes
	if rel.Info != nil && rel.Info.Notes != "" {
		notesCard := component.NewCard(component.TitleFromString("Release Notes"))
		notesCard.SetBody(component.NewText(rel.Info.Notes))
		components = append(components, notesCard)
	}

	// Rendered manifest
	if rel.Manifest != "" {
		manifestCard := component.NewCard(component.TitleFromString("Rendered Manifest"))
		manifestCard.SetBody(component.NewText(rel.Manifest))
		components = append(components, manifestCard)
	}

	// K8s resource links parsed from manifest
	if resourceTable := buildResourceTable(rel.Manifest, relNamespace); resourceTable != nil {
		components = append(components, resourceTable)
	}

	// History table with rollback buttons
	history, err := client.getReleaseHistory(relName)
	if err == nil {
		components = append(components, buildHistoryTable(history, relName, relNamespace, rel.Version))
	}

	return component.ContentResponse{
		Title:      component.TitleFromString(fmt.Sprintf("Helm Release: %s", relName)),
		Components: components,
	}, nil
}

func buildSummaryCard(rel *helmRelease.Release, relName, relNamespace string) component.Component {
	var sections []component.SummarySection
	if rel.Info != nil {
		sections = append(sections,
			component.SummarySection{Header: "Status", Content: component.NewText(string(rel.Info.Status))},
			component.SummarySection{Header: "Last Deployed", Content: component.NewText(rel.Info.LastDeployed.String())},
		)
		if rel.Info.Description != "" && rel.Info.Status != helmRelease.StatusFailed {
			sections = append(sections, component.SummarySection{Header: "Description", Content: component.NewText(rel.Info.Description)})
		}
	}
	sections = append(sections,
		component.SummarySection{Header: "Namespace", Content: component.NewText(relNamespace)},
		component.SummarySection{Header: "Revision", Content: component.NewText(fmt.Sprintf("%d", rel.Version))},
	)
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		sections = append(sections,
			component.SummarySection{Header: "Chart", Content: component.NewText(fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version))},
			component.SummarySection{Header: "App Version", Content: component.NewText(rel.Chart.Metadata.AppVersion)},
		)
	}
	return component.NewSummary(fmt.Sprintf("Release: %s", relName), sections...)
}

func buildValuesCard(client *helmClient, relName, relNamespace string) component.Component {
	card := component.NewCard(component.TitleFromString("Values (computed)"))
	values, err := client.getReleaseValues(relName, true)
	if err != nil {
		card.SetBody(component.NewText(fmt.Sprintf("Unable to load values: %v", err)))
		return card
	}
	if len(values) == 0 {
		card.SetBody(component.NewText("(no values set)"))
		return card
	}
	out, err := yaml.Marshal(values)
	if err != nil {
		card.SetBody(component.NewText(fmt.Sprintf("Unable to render values: %v", err)))
		return card
	}
	card.SetBody(component.NewText(string(out)))
	return card
}

func buildUpgradeCard(client *helmClient, relName, relNamespace string) component.Component {
	card := component.NewCard(component.TitleFromString("Upgrade Release"))

	// Load user-supplied values as the starting point for the editor
	userValues, err := client.getReleaseValues(relName, false)
	if err != nil {
		userValues = map[string]interface{}{}
	}
	valuesYAML := ""
	if len(userValues) > 0 {
		if out, err := yaml.Marshal(userValues); err == nil {
			valuesYAML = string(out)
		}
	}

	upgradePayload := action.Payload{
		"action":    ActionHelmUpgrade,
		"name":      relName,
		"namespace": relNamespace,
		"values":    valuesYAML,
	}
	upgradeBtn := component.NewButton("Upgrade with current values", upgradePayload,
		component.WithButtonConfirmation(
			"Upgrade Release",
			fmt.Sprintf("Upgrade release %q with the current values? Edit the values YAML above before confirming.", relName),
		),
	)
	btnGroup := component.NewButtonGroup()
	btnGroup.AddButton(upgradeBtn)
	card.SetBody(btnGroup)
	return card
}

func buildHistoryTable(history []*helmRelease.Release, relName, relNamespace string, currentVersion int) component.Component {
	cols := component.NewTableCols("Revision", "Updated", "Status", "Chart", "App Version", "Description", "Actions")
	table := component.NewTable("Revision History", "No history available", cols)

	for _, h := range history {
		if h.Info == nil {
			continue
		}
		chartName, appVersion := "", ""
		if h.Chart != nil && h.Chart.Metadata != nil {
			chartName = fmt.Sprintf("%s-%s", h.Chart.Metadata.Name, h.Chart.Metadata.Version)
			appVersion = h.Chart.Metadata.AppVersion
		}

		var actionsCell component.Component = component.NewText("")
		if h.Version != currentVersion {
			rollbackPayload := action.Payload{
				"action":    ActionHelmRollback,
				"name":      relName,
				"namespace": relNamespace,
				"revision":  int64(h.Version),
			}
			rollbackBtn := component.NewButton(fmt.Sprintf("Rollback to %d", h.Version), rollbackPayload,
				component.WithButtonConfirmation(
					"Rollback Release",
					fmt.Sprintf("Roll back release %q to revision %d?", relName, h.Version),
				),
			)
			bg := component.NewButtonGroup()
			bg.AddButton(rollbackBtn)
			actionsCell = bg
		}

		table.Add(component.TableRow{
			"Revision":    component.NewText(fmt.Sprintf("%d", h.Version)),
			"Updated":     component.NewText(h.Info.LastDeployed.String()),
			"Status":      component.NewText(string(h.Info.Status)),
			"Chart":       component.NewText(chartName),
			"App Version": component.NewText(appVersion),
			"Description": component.NewText(h.Info.Description),
			"Actions":     actionsCell,
		})
	}
	return table
}

// buildResourceTable parses the release manifest YAML and returns a table of
// K8s resources with links to their Octant detail pages.
func buildResourceTable(manifest, relNamespace string) *component.Table {
	type resourceDoc struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace"`
		} `yaml:"metadata"`
	}

	if strings.TrimSpace(manifest) == "" {
		return nil
	}

	decoder := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	var resources []resourceDoc
	for {
		var doc resourceDoc
		if err := decoder.Decode(&doc); err != nil {
			break
		}
		if doc.Kind != "" && doc.Metadata.Name != "" {
			if doc.Metadata.Namespace == "" {
				doc.Metadata.Namespace = relNamespace
			}
			resources = append(resources, doc)
		}
	}

	if len(resources) == 0 {
		return nil
	}

	cols := component.NewTableCols("Kind", "Name", "Namespace")
	table := component.NewTable("Deployed Resources", "No resources found", cols)
	for _, r := range resources {
		octantPath := octantResourcePath(r.APIVersion, r.Kind, r.Metadata.Namespace, r.Metadata.Name)
		var nameCell component.Component
		if octantPath != "" {
			nameCell = component.NewLink("", r.Metadata.Name, octantPath)
		} else {
			nameCell = component.NewText(r.Metadata.Name)
		}
		table.Add(component.TableRow{
			"Kind":      component.NewText(r.Kind),
			"Name":      nameCell,
			"Namespace": component.NewText(r.Metadata.Namespace),
		})
	}
	return table
}

// octantResourcePath returns the Octant URL path for a known K8s resource kind.
func octantResourcePath(apiVersion, kind, namespace, name string) string {
	base := fmt.Sprintf("/namespace/%s", namespace)
	switch kind {
	case "Deployment":
		return fmt.Sprintf("%s/workloads/deployments/%s", base, name)
	case "StatefulSet":
		return fmt.Sprintf("%s/workloads/stateful-sets/%s", base, name)
	case "DaemonSet":
		return fmt.Sprintf("%s/workloads/daemon-sets/%s", base, name)
	case "ReplicaSet":
		return fmt.Sprintf("%s/workloads/replica-sets/%s", base, name)
	case "CronJob":
		return fmt.Sprintf("%s/workloads/cron-jobs/%s", base, name)
	case "Job":
		return fmt.Sprintf("%s/workloads/jobs/%s", base, name)
	case "Pod":
		return fmt.Sprintf("%s/workloads/pods/%s", base, name)
	case "Service":
		return fmt.Sprintf("%s/discovery-and-load-balancing/services/%s", base, name)
	case "Ingress":
		return fmt.Sprintf("%s/discovery-and-load-balancing/ingresses/%s", base, name)
	case "ConfigMap":
		return fmt.Sprintf("%s/config-and-storage/config-maps/%s", base, name)
	case "Secret":
		return fmt.Sprintf("%s/config-and-storage/secrets/%s", base, name)
	case "PersistentVolumeClaim":
		return fmt.Sprintf("%s/config-and-storage/persistent-volume-claims/%s", base, name)
	case "ServiceAccount":
		return fmt.Sprintf("%s/config-and-storage/service-accounts/%s", base, name)
	case "Role", "ClusterRole", "RoleBinding", "ClusterRoleBinding":
		return fmt.Sprintf("%s/rbac/%s/%s", base, strings.ToLower(kind)+"s", name)
	case "HorizontalPodAutoscaler":
		return fmt.Sprintf("%s/discovery-and-load-balancing/horizontal-pod-autoscalers/%s", base, name)
	case "PodDisruptionBudget":
		return fmt.Sprintf("%s/workloads/pod-disruption-budgets/%s", base, name)
	}
	return ""
}

// PathFilters returns path filters for individual release detail pages.
func (d *releaseDescriber) PathFilters() []describer.PathFilter {
	return []describer.PathFilter{
		*describer.NewPathFilter("/:namespace/:name", d),
	}
}

// Reset does nothing.
func (d *releaseDescriber) Reset(ctx context.Context) error { return nil }

func parseReleasePath(fields map[string]string) (namespace, name string, err error) {
	name = strings.TrimPrefix(fields["name"], "/")
	namespace = strings.TrimPrefix(fields["namespace"], "/")
	if name == "" || namespace == "" {
		return "", "", fmt.Errorf("missing release name or namespace in path (fields: %v)", fields)
	}
	return namespace, name, nil
}