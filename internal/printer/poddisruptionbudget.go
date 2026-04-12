/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package printer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	policyv1 "k8s.io/api/policy/v1"

	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// PodDisruptionBudgetListHandler is a printFunc that prints PodDisruptionBudgets
func PodDisruptionBudgetListHandler(ctx context.Context, list *policyv1.PodDisruptionBudgetList, options Options) (component.Component, error) {
	if list == nil {
		return nil, errors.New("pod disruption budget list is nil")
	}

	cols := component.NewTableCols("Name", "Min Available", "Max Unavailable", "Allowed Disruptions", "Age")
	ot := NewObjectTable("Pod Disruption Budgets", "We couldn't find any pod disruption budgets!", cols, options.DashConfig.ObjectStore())
	ot.EnablePluginStatus(options.DashConfig.PluginManager())

	for _, pdb := range list.Items {
		row := component.TableRow{}

		nameLink, err := options.Link.ForObject(&pdb, pdb.Name)
		if err != nil {
			return nil, err
		}

		minAvailable := "<none>"
		if pdb.Spec.MinAvailable != nil {
			minAvailable = pdb.Spec.MinAvailable.String()
		}

		maxUnavailable := "<none>"
		if pdb.Spec.MaxUnavailable != nil {
			maxUnavailable = pdb.Spec.MaxUnavailable.String()
		}

		allowedDisruptions := fmt.Sprintf("%d", pdb.Status.DisruptionsAllowed)

		row["Name"] = nameLink
		row["Min Available"] = component.NewText(minAvailable)
		row["Max Unavailable"] = component.NewText(maxUnavailable)
		row["Allowed Disruptions"] = component.NewText(allowedDisruptions)
		row["Age"] = component.NewTimestamp(pdb.CreationTimestamp.Time)

		if err := ot.AddRowForObject(ctx, &pdb, row); err != nil {
			return nil, fmt.Errorf("add row for object: %w", err)
		}
	}

	return ot.ToComponent()
}

// PodDisruptionBudgetHandler is a printFunc that prints a PodDisruptionBudget
func PodDisruptionBudgetHandler(ctx context.Context, pdb *policyv1.PodDisruptionBudget, options Options) (component.Component, error) {
	o := NewObject(pdb)
	o.EnableEvents()

	var sections component.SummarySections

	if pdb.Spec.MinAvailable != nil {
		sections.AddText("Min Available", pdb.Spec.MinAvailable.String())
	}
	if pdb.Spec.MaxUnavailable != nil {
		sections.AddText("Max Unavailable", pdb.Spec.MaxUnavailable.String())
	}
	if pdb.Spec.Selector != nil {
		sections.Add("Selector", printSelector(pdb.Spec.Selector))
	}

	configSummary := component.NewSummary("Configuration", sections...)
	o.RegisterConfig(configSummary)

	var statusSections component.SummarySections
	statusSections.AddText("Disruptions Allowed", fmt.Sprintf("%d", pdb.Status.DisruptionsAllowed))
	statusSections.AddText("Current Healthy", fmt.Sprintf("%d", pdb.Status.CurrentHealthy))
	statusSections.AddText("Desired Healthy", fmt.Sprintf("%d", pdb.Status.DesiredHealthy))
	statusSections.AddText("Expected Pods", fmt.Sprintf("%d", pdb.Status.ExpectedPods))
	statusSummary := component.NewSummary("Status", statusSections...)
	o.RegisterSummary(statusSummary)

	return o.ToComponent(ctx, options)
}