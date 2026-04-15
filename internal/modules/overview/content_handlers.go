/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package overview

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/utils/clock"
)

// translateTimestamp returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestamp(timestamp metav1.Time, c clock.Clock) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.ShortHumanDuration(c.Since(timestamp.Time))
}
