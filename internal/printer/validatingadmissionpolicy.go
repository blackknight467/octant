/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"

	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// ValidatingAdmissionPolicyListHandler is a printFunc that prints ValidatingAdmissionPolicies
func ValidatingAdmissionPolicyListHandler(ctx context.Context, list *admissionregistrationv1alpha1.ValidatingAdmissionPolicyList, options Options) (component.Component, error) {
	if list == nil {
		return nil, errors.New("validating admission policy list is nil")
	}

	cols := component.NewTableCols("Name", "Validations", "Age")
	ot := NewObjectTable("Validating Admission Policies", "We couldn't find any validating admission policies!", cols, options.DashConfig.ObjectStore())
	ot.EnablePluginStatus(options.DashConfig.PluginManager())

	for _, vap := range list.Items {
		row := component.TableRow{}

		nameLink, err := options.Link.ForObject(&vap, vap.Name)
		if err != nil {
			return nil, err
		}

		row["Name"] = nameLink
		row["Validations"] = component.NewText(fmt.Sprintf("%d", len(vap.Spec.Validations)))
		row["Age"] = component.NewTimestamp(vap.CreationTimestamp.Time)

		if err := ot.AddRowForObject(ctx, &vap, row); err != nil {
			return nil, fmt.Errorf("add row for object: %w", err)
		}
	}

	return ot.ToComponent()
}

// ValidatingAdmissionPolicyHandler is a printFunc that prints a ValidatingAdmissionPolicy
func ValidatingAdmissionPolicyHandler(ctx context.Context, vap *admissionregistrationv1alpha1.ValidatingAdmissionPolicy, options Options) (component.Component, error) {
	o := NewObject(vap)
	o.EnableEvents()

	var sections component.SummarySections

	if vap.Spec.FailurePolicy != nil {
		sections.AddText("Failure Policy", string(*vap.Spec.FailurePolicy))
	}

	if vap.Spec.MatchConstraints != nil {
		mc := vap.Spec.MatchConstraints
		if mc.ResourceRules != nil {
			var ruleStrs []string
			for _, r := range mc.ResourceRules {
				ruleStrs = append(ruleStrs, fmt.Sprintf("%s/%s: %s",
					strings.Join(r.APIGroups, ","),
					strings.Join(r.APIVersions, ","),
					strings.Join(r.Resources, ",")))
			}
			sections.AddText("Match Resources", strings.Join(ruleStrs, "; "))
		}
	}

	configSummary := component.NewSummary("Configuration", sections...)
	o.RegisterConfig(configSummary)

	if len(vap.Spec.Validations) > 0 {
		cols := component.NewTableCols("Expression", "Message", "Reason")
		validationTable := component.NewTable("Validations", "No validations found", cols)
		for _, v := range vap.Spec.Validations {
			row := component.TableRow{}
			row["Expression"] = component.NewText(v.Expression)
			msg := ""
			if v.Message != "" {
				msg = v.Message
			}
			row["Message"] = component.NewText(msg)
			reason := ""
			if v.Reason != nil {
				reason = string(*v.Reason)
			}
			row["Reason"] = component.NewText(reason)
			validationTable.Add(row)
		}

		o.RegisterItems(ItemDescriptor{
			Width: component.WidthFull,
			Func: func() (component.Component, error) {
				return validationTable, nil
			},
		})
	}

	return o.ToComponent(ctx, options)
}
