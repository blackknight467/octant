/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"

	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/config"
)

const (
	// ActionHelmUninstall is the action name for uninstalling a Helm release.
	ActionHelmUninstall = "helm.octant.dev/uninstall"
)

// uninstallAction handles Helm release uninstallation.
type uninstallAction struct {
	dashConfig config.Dash
}

var _ action.Dispatcher = (*uninstallAction)(nil)

func newUninstallAction(dashConfig config.Dash) *uninstallAction {
	return &uninstallAction{dashConfig: dashConfig}
}

// ActionName returns the action name.
func (a *uninstallAction) ActionName() string {
	return ActionHelmUninstall
}

// Handle processes the uninstall action.
func (a *uninstallAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, err := payload.String("name")
	if err != nil {
		return fmt.Errorf("payload missing name: %w", err)
	}
	namespace, err := payload.String("namespace")
	if err != nil {
		return fmt.Errorf("payload missing namespace: %w", err)
	}

	client, err := newHelmClient(a.dashConfig, namespace)
	if err != nil {
		return fmt.Errorf("helm client: %w", err)
	}

	if _, err := client.uninstallRelease(name); err != nil {
		alerter.SendAlert(action.Alert{
			Type:    action.AlertTypeError,
			Message: fmt.Sprintf("Failed to uninstall release %q: %v", name, err),
		})
		return err
	}

	alerter.SendAlert(action.Alert{
		Type:    action.AlertTypeSuccess,
		Message: fmt.Sprintf("Release %q uninstalled successfully", name),
	})
	return nil
}