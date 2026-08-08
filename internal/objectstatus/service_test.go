/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package objectstatus

import (
	"context"
	"testing"

	linkFake "github.com/vmware-tanzu/octant/internal/link/fake"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/vmware-tanzu/octant/internal/testutil"
	"github.com/vmware-tanzu/octant/pkg/store"
	storefake "github.com/vmware-tanzu/octant/pkg/store/fake"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

func Test_service(t *testing.T) {
	cases := []struct {
		name     string
		init     func(*testing.T, *storefake.MockStore) runtime.Object
		expected ObjectStatus
		isErr    bool
	}{
		{
			name: "in general",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				selector := labels.Set{discoveryv1.LabelServiceName: "stateful"}
				key := store.Key{
					Namespace:  "default",
					APIVersion: "discovery.k8s.io/v1",
					Kind:       "EndpointSlice",
					Selector:   &selector,
				}

				endpointSlice := testutil.LoadObjectFromFile(t, "endpointslice_ok.yaml")

				o.EXPECT().List(gomock.Any(), gomock.Eq(key)).
					Return(testutil.ToUnstructuredList(t, endpointSlice), false, nil)

				objectFile := "service_ok.yaml"
				return testutil.LoadObjectFromFile(t, objectFile)

			},
			expected: ObjectStatus{
				NodeStatus: component.NodeStatusOK,
				Details:    []component.Component{component.NewText("Service is OK")},
				Properties: []component.Property{{Label: "Type", Value: component.NewText("ClusterIP")},
					{Label: "Session Affinity", Value: component.NewText("None")}},
			},
		},
		{
			name: "externalName",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				objectFile := "service_external.yaml"
				return testutil.LoadObjectFromFile(t, objectFile)
			},
			expected: ObjectStatus{
				NodeStatus: component.NodeStatusOK,
				Details:    []component.Component{component.NewText("Service is OK")},
				Properties: []component.Property{{Label: "Type", Value: component.NewText("ExternalName")},
					{Label: "Session Affinity", Value: component.NewText("")}},
			},
		},
		{
			name: "no endpoint subsets",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				selector := labels.Set{discoveryv1.LabelServiceName: "stateful"}
				key := store.Key{
					Namespace:  "default",
					APIVersion: "discovery.k8s.io/v1",
					Kind:       "EndpointSlice",
					Selector:   &selector,
				}

				endpointSlice := testutil.LoadObjectFromFile(t, "endpointslice_no_endpoints.yaml")

				o.EXPECT().List(gomock.Any(), gomock.Eq(key)).
					Return(testutil.ToUnstructuredList(t, endpointSlice), false, nil)

				objectFile := "service_ok.yaml"
				return testutil.LoadObjectFromFile(t, objectFile)

			},
			expected: ObjectStatus{
				NodeStatus: component.NodeStatusWarning,
				Details:    []component.Component{component.NewText("Service has no endpoint addresses")},
			},
		},
		{
			// With v1 Endpoints, "ready" was structural (Addresses vs
			// NotReadyAddresses). On EndpointSlice it is a hand-written condition,
			// so it needs its own coverage.
			name: "endpoints exist but none are ready",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				selector := labels.Set{discoveryv1.LabelServiceName: "stateful"}
				key := store.Key{
					Namespace:  "default",
					APIVersion: "discovery.k8s.io/v1",
					Kind:       "EndpointSlice",
					Selector:   &selector,
				}

				endpointSlice := testutil.LoadObjectFromFile(t, "endpointslice_not_ready.yaml")

				o.EXPECT().List(gomock.Any(), gomock.Eq(key)).
					Return(testutil.ToUnstructuredList(t, endpointSlice), false, nil)

				return testutil.LoadObjectFromFile(t, "service_ok.yaml")
			},
			expected: ObjectStatus{
				NodeStatus: component.NodeStatusWarning,
				Details:    []component.Component{component.NewText("Service has no endpoint addresses")},
			},
		},
		{
			// A nil ready condition means unknown, which must be read as ready.
			name: "endpoint with no ready condition counts as ready",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				selector := labels.Set{discoveryv1.LabelServiceName: "stateful"}
				key := store.Key{
					Namespace:  "default",
					APIVersion: "discovery.k8s.io/v1",
					Kind:       "EndpointSlice",
					Selector:   &selector,
				}

				endpointSlice := testutil.LoadObjectFromFile(t, "endpointslice_nil_ready.yaml")

				o.EXPECT().List(gomock.Any(), gomock.Eq(key)).
					Return(testutil.ToUnstructuredList(t, endpointSlice), false, nil)

				return testutil.LoadObjectFromFile(t, "service_ok.yaml")
			},
			expected: ObjectStatus{
				NodeStatus: component.NodeStatusOK,
				Details:    []component.Component{component.NewText("Service is OK")},
				Properties: []component.Property{{Label: "Type", Value: component.NewText("ClusterIP")},
					{Label: "Session Affinity", Value: component.NewText("None")}},
			},
		},
		{
			name: "object is nil",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				return nil
			},
			isErr: true,
		},
		{
			name: "object is not a daemon set",
			init: func(t *testing.T, o *storefake.MockStore) runtime.Object {
				return &unstructured.Unstructured{}
			},
			isErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			linkInterface := linkFake.NewMockInterface(controller)
			defer controller.Finish()

			o := storefake.NewMockStore(controller)

			object := tc.init(t, o)

			ctx := context.Background()
			status, err := service(ctx, object, o, linkInterface)
			if tc.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expected, status)
		})
	}
}
