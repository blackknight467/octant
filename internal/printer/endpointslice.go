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
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// EndpointSliceListHandler is a printFunc that prints EndpointSlices
func EndpointSliceListHandler(ctx context.Context, list *discoveryv1.EndpointSliceList, options Options) (component.Component, error) {
	if list == nil {
		return nil, errors.New("endpoint slice list is nil")
	}

	cols := component.NewTableCols("Name", "Address Type", "Ports", "Endpoints", "Age")
	ot := NewObjectTable("Endpoint Slices", "We couldn't find any endpoint slices!", cols, options.DashConfig.ObjectStore())
	ot.EnablePluginStatus(options.DashConfig.PluginManager())

	for _, es := range list.Items {
		row := component.TableRow{}

		nameLink, err := options.Link.ForObject(&es, es.Name)
		if err != nil {
			return nil, err
		}

		var portNames []string
		for _, p := range es.Ports {
			if p.Port != nil {
				portStr := fmt.Sprintf("%d", *p.Port)
				if p.Protocol != nil {
					portStr = string(*p.Protocol) + ":" + portStr
				}
				portNames = append(portNames, portStr)
			}
		}
		portsStr := strings.Join(portNames, ", ")
		if portsStr == "" {
			portsStr = "<none>"
		}

		row["Name"] = nameLink
		row["Address Type"] = component.NewText(string(es.AddressType))
		row["Ports"] = component.NewText(portsStr)
		row["Endpoints"] = component.NewText(fmt.Sprintf("%d", len(es.Endpoints)))
		row["Age"] = component.NewTimestamp(es.CreationTimestamp.Time)

		if err := ot.AddRowForObject(ctx, &es, row); err != nil {
			return nil, fmt.Errorf("add row for object: %w", err)
		}
	}

	return ot.ToComponent()
}

// EndpointSliceHandler is a printFunc that prints an EndpointSlice
func EndpointSliceHandler(ctx context.Context, es *discoveryv1.EndpointSlice, options Options) (component.Component, error) {
	o := NewObject(es)
	o.EnableEvents()

	var sections component.SummarySections
	sections.AddText("Address Type", string(es.AddressType))

	var portStrs []string
	for _, p := range es.Ports {
		parts := []string{}
		if p.Name != nil && *p.Name != "" {
			parts = append(parts, *p.Name)
		}
		if p.Protocol != nil {
			parts = append(parts, string(*p.Protocol))
		}
		if p.Port != nil {
			parts = append(parts, fmt.Sprintf("%d", *p.Port))
		}
		portStrs = append(portStrs, strings.Join(parts, "/"))
	}
	if len(portStrs) > 0 {
		sections.AddText("Ports", strings.Join(portStrs, ", "))
	}

	configSummary := component.NewSummary("Configuration", sections...)
	o.RegisterConfig(configSummary)

	cols := component.NewTableCols("Addresses", "Conditions", "Node", "Zone")
	endpointTable := component.NewTable("Endpoints", "No endpoints found", cols)
	for _, ep := range es.Endpoints {
		row := component.TableRow{}

		row["Addresses"] = component.NewText(strings.Join(ep.Addresses, ", "))

		conditions := []string{}
		if ep.Conditions.Ready != nil {
			if *ep.Conditions.Ready {
				conditions = append(conditions, "ready")
			} else {
				conditions = append(conditions, "not ready")
			}
		}
		if ep.Conditions.Serving != nil && *ep.Conditions.Serving {
			conditions = append(conditions, "serving")
		}
		if ep.Conditions.Terminating != nil && *ep.Conditions.Terminating {
			conditions = append(conditions, "terminating")
		}
		row["Conditions"] = component.NewText(strings.Join(conditions, ", "))

		nodeName := "<none>"
		if ep.NodeName != nil {
			nodeName = *ep.NodeName
		}
		row["Node"] = component.NewText(nodeName)

		zone := "<none>"
		if ep.Zone != nil {
			zone = *ep.Zone
		}
		row["Zone"] = component.NewText(zone)

		endpointTable.Add(row)
	}

	o.RegisterItems(ItemDescriptor{
		Width: component.WidthFull,
		Func: func() (component.Component, error) {
			return endpointTable, nil
		},
	})

	return o.ToComponent(ctx, options)
}