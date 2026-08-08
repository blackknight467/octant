/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/config"
)

const (
	ActionHelmUninstall  = "helm.octant.dev/uninstall"
	ActionHelmRollback   = "helm.octant.dev/rollback"
	ActionHelmUpgrade    = "helm.octant.dev/upgrade"
	ActionHelmTest       = "helm.octant.dev/test"
	ActionHelmRemoveRepo = "helm.octant.dev/remove-repo"
)

// --- Uninstall ---

type uninstallAction struct{ dashConfig config.Dash }

var _ action.Dispatcher = (*uninstallAction)(nil)

func newUninstallAction(dashConfig config.Dash) *uninstallAction {
	return &uninstallAction{dashConfig: dashConfig}
}
func (a *uninstallAction) ActionName() string { return ActionHelmUninstall }
func (a *uninstallAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, _ := payload.String("name")
	namespace, _ := payload.String("namespace")
	client, err := newHelmClient(a.dashConfig, namespace)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("helm client error: %v", err))
	}
	if _, err := client.uninstallRelease(name); err != nil {
		return sendError(alerter, fmt.Sprintf("Failed to uninstall %q: %v", name, err))
	}
	alerter.SendAlert(action.Alert{Type: action.AlertTypeSuccess, Message: fmt.Sprintf("Release %q uninstalled", name)})
	return nil
}

// --- Rollback ---

type rollbackAction struct{ dashConfig config.Dash }

var _ action.Dispatcher = (*rollbackAction)(nil)

func newRollbackAction(dashConfig config.Dash) *rollbackAction {
	return &rollbackAction{dashConfig: dashConfig}
}
func (a *rollbackAction) ActionName() string { return ActionHelmRollback }
func (a *rollbackAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, _ := payload.String("name")
	namespace, _ := payload.String("namespace")
	revision, err := payload.Int64("revision")
	if err != nil {
		return sendError(alerter, fmt.Sprintf("invalid revision: %v", err))
	}
	client, err := newHelmClient(a.dashConfig, namespace)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("helm client error: %v", err))
	}
	if err := client.rollbackRelease(name, int(revision)); err != nil {
		return sendError(alerter, fmt.Sprintf("Failed to rollback %q to revision %d: %v", name, revision, err))
	}
	alerter.SendAlert(action.Alert{Type: action.AlertTypeSuccess, Message: fmt.Sprintf("Release %q rolled back to revision %d", name, revision)})
	return nil
}

// --- Upgrade ---

type upgradeAction struct{ dashConfig config.Dash }

var _ action.Dispatcher = (*upgradeAction)(nil)

func newUpgradeAction(dashConfig config.Dash) *upgradeAction {
	return &upgradeAction{dashConfig: dashConfig}
}
func (a *upgradeAction) ActionName() string { return ActionHelmUpgrade }
func (a *upgradeAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, _ := payload.String("name")
	namespace, _ := payload.String("namespace")
	valuesYAML, _ := payload.String("values")

	var values map[string]interface{}
	if valuesYAML != "" {
		if err := yaml.Unmarshal([]byte(valuesYAML), &values); err != nil {
			return sendError(alerter, fmt.Sprintf("invalid YAML values: %v", err))
		}
	}

	client, err := newHelmClient(a.dashConfig, namespace)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("helm client error: %v", err))
	}
	rel, err := client.upgradeRelease(name, values)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("Failed to upgrade %q: %v", name, err))
	}
	alerter.SendAlert(action.Alert{Type: action.AlertTypeSuccess, Message: fmt.Sprintf("Release %q upgraded to revision %d", name, rel.Version)})
	return nil
}

// --- Test ---

type testAction struct{ dashConfig config.Dash }

var _ action.Dispatcher = (*testAction)(nil)

func newTestAction(dashConfig config.Dash) *testAction {
	return &testAction{dashConfig: dashConfig}
}
func (a *testAction) ActionName() string { return ActionHelmTest }
func (a *testAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, _ := payload.String("name")
	namespace, _ := payload.String("namespace")
	client, err := newHelmClient(a.dashConfig, namespace)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("helm client error: %v", err))
	}
	rel, err := client.runTests(name)
	if err != nil {
		return sendError(alerter, fmt.Sprintf("Tests failed for %q: %v", name, err))
	}
	msg := fmt.Sprintf("Tests passed for %q", name)
	if rel != nil && rel.Info != nil && rel.Info.Notes != "" {
		msg += ": " + rel.Info.Notes
	}
	alerter.SendAlert(action.Alert{Type: action.AlertTypeSuccess, Message: msg})
	return nil
}

// --- Remove repo ---

type removeRepoAction struct{ dashConfig config.Dash }

var _ action.Dispatcher = (*removeRepoAction)(nil)

func newRemoveRepoAction(dashConfig config.Dash) *removeRepoAction {
	return &removeRepoAction{dashConfig: dashConfig}
}
func (a *removeRepoAction) ActionName() string { return ActionHelmRemoveRepo }
func (a *removeRepoAction) Handle(ctx context.Context, alerter action.Alerter, payload action.Payload) error {
	name, _ := payload.String("name")
	if err := removeRepository(name); err != nil {
		return sendError(alerter, fmt.Sprintf("Failed to remove repo %q: %v", name, err))
	}
	alerter.SendAlert(action.Alert{Type: action.AlertTypeSuccess, Message: fmt.Sprintf("Repository %q removed", name)})
	return nil
}

// --- helpers ---

func sendError(alerter action.Alerter, msg string) error {
	alerter.SendAlert(action.Alert{Type: action.AlertTypeError, Message: msg})
	return fmt.Errorf("%s", msg)
}
