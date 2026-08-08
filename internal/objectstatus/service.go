/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package objectstatus

import (
	"context"
	goerrors "errors"

	"github.com/vmware-tanzu/octant/internal/link"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	internalErrors "github.com/vmware-tanzu/octant/internal/errors"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
	"github.com/vmware-tanzu/octant/pkg/store"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

func service(ctx context.Context, object runtime.Object, o store.Store, _ link.Interface) (ObjectStatus, error) {
	if object == nil {
		return ObjectStatus{}, errors.Errorf("service is nil")
	}

	service := &corev1.Service{}

	if err := scheme.Scheme.Convert(object, service, 0); err != nil {
		return ObjectStatus{}, errors.Wrap(err, "convert object to service")
	}

	if service.Spec.ExternalName == "" {
		// A service's endpoints live in one or more EndpointSlices labelled
		// with the service name. v1 Endpoints is deprecated as of k8s 1.33.
		selector := labels.Set{discoveryv1.LabelServiceName: service.Name}
		key := store.Key{
			Namespace:  service.Namespace,
			APIVersion: "discovery.k8s.io/v1",
			Kind:       "EndpointSlice",
			Selector:   &selector,
		}

		list, _, err := o.List(ctx, key)
		if err != nil {
			// Roles predating EndpointSlice commonly grant core/v1 endpoints but not
			// discovery.k8s.io/v1 endpointslices. Returning an error here would fail
			// the whole Services table render, so report the status as unknown and
			// leave the rest of the list intact.
			var ae *internalErrors.AccessError
			if goerrors.As(err, &ae) {
				return ObjectStatus{
					NodeStatus: component.NodeStatusWarning,
					Details: []component.Component{
						component.NewText("Endpoint status unavailable: no access to endpointslices"),
					},
				}, nil
			}
			return ObjectStatus{}, errors.Wrapf(err, "list endpoint slices for service %s", service.Name)
		}

		if list == nil || len(list.Items) == 0 {
			return ObjectStatus{
				NodeStatus: component.NodeStatusWarning,
				Details:    []component.Component{component.NewText("Service has no endpoints")},
			}, nil
		}

		addressCount := 0

		for i := range list.Items {
			endpointSlice := &discoveryv1.EndpointSlice{}
			if err := scheme.Scheme.Convert(&list.Items[i], endpointSlice, 0); err != nil {
				return ObjectStatus{}, errors.Wrap(err, "convert unstructured object to endpoint slice")
			}

			for _, endpoint := range endpointSlice.Endpoints {
				if kubernetes.EndpointReady(endpoint) {
					addressCount += len(endpoint.Addresses)
				}
			}
		}

		if addressCount == 0 {
			return ObjectStatus{
				NodeStatus: component.NodeStatusWarning,
				Details:    []component.Component{component.NewText("Service has no endpoint addresses")},
			}, nil
		}
	}
	properties := []component.Property{{Label: "Type", Value: component.NewText(string(service.Spec.Type))},
		{Label: "Session Affinity", Value: component.NewText(string(service.Spec.SessionAffinity))}}

	return ObjectStatus{
		NodeStatus: component.NodeStatusOK,
		Details:    []component.Component{component.NewText("Service is OK")},
		Properties: properties,
	}, nil
}
