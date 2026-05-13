// Copyright (c) 2024 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package octant

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/vmware-tanzu/octant/internal/log"
	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/cluster"
	"github.com/vmware-tanzu/octant/pkg/store"
)

// RolloutRestart triggers a rollout restart on a Deployment or DaemonSet.
type RolloutRestart struct {
	clusterClient func() cluster.ClientInterface
}

var _ action.Dispatcher = (*RolloutRestart)(nil)

func NewRolloutRestart(clusterClient func() cluster.ClientInterface) *RolloutRestart {
	return &RolloutRestart{clusterClient: clusterClient}
}

func (r *RolloutRestart) ActionName() string {
	return ActionOverviewRolloutRestart
}

func (r *RolloutRestart) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	logger := log.From(ctx).With("actionName", r.ActionName())
	logger.With("payload", payload).Infof("received action payload")

	key, err := store.KeyFromPayload(payload)
	if err != nil {
		return err
	}

	client, err := r.clusterClient().KubernetesClient()
	if err != nil {
		return err
	}

	restartedAt := time.Now().UTC().Format(time.RFC3339)
	patch := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		restartedAt,
	)

	var alertMsg string
	alertType := action.AlertTypeInfo

	switch key.Kind {
	case "Deployment":
		_, err = client.AppsV1().Deployments(key.Namespace).Patch(
			ctx, key.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{},
		)
		alertMsg = fmt.Sprintf("Rollout restart triggered for Deployment %q", key.Name)
	case "DaemonSet":
		_, err = client.AppsV1().DaemonSets(key.Namespace).Patch(
			ctx, key.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{},
		)
		alertMsg = fmt.Sprintf("Rollout restart triggered for DaemonSet %q", key.Name)
	default:
		return fmt.Errorf("rollout restart not supported for kind %q", key.Kind)
	}

	if err != nil {
		alertMsg = fmt.Sprintf("Failed to restart %s %q: %s", key.Kind, key.Name, err)
		alertType = action.AlertTypeWarning
		logger.WithErr(err).Errorf("rollout restart")
	}

	alerter.SendAlert(action.CreateAlert(alertType, alertMsg, action.DefaultAlertExpiration))
	return nil
}