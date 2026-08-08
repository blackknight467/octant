/*
Copyright (c) 2026 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package kubernetes

import (
	discoveryv1 "k8s.io/api/discovery/v1"
)

// EndpointReady reports whether an EndpointSlice endpoint should be treated as
// serving traffic.
//
// A nil Ready condition means "unknown", which consumers are expected to read as
// ready. That matches what v1 Endpoints placed in Subsets.Addresses (as opposed to
// Subsets.NotReadyAddresses), so callers migrating off v1 Endpoints keep the same
// ready-address semantics.
func EndpointReady(endpoint discoveryv1.Endpoint) bool {
	return endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready
}
