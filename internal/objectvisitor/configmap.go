/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package objectvisitor

import (
	"context"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/octant/internal/gvk"
	"github.com/vmware-tanzu/octant/internal/queryer"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
)

// ConfigMap is a typed visitor for config maps.
type ConfigMap struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*ConfigMap)(nil)

// NewConfigMap creates an instance of ConfigMap.
func NewConfigMap(q queryer.Queryer) *ConfigMap {
	return &ConfigMap{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (ConfigMap) Supports() schema.GroupVersionKind {
	return gvk.ConfigMap
}

// Visit visits a config map. It links to pods that reference it.
func (c *ConfigMap) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitConfigMap")
	defer span.End()

	configMap := &corev1.ConfigMap{}
	if err := kubernetes.FromUnstructured(object, configMap); err != nil {
		return err
	}
	handler.SetLevel(configMap.Kind, level)

	var g errgroup.Group

	g.Go(func() error {
		pods, err := c.queryer.PodsForConfigMap(ctx, configMap)
		if err != nil {
			return err
		}
		for i := range pods {
			pod := pods[i]
			g.Go(func() error {
				m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
				if err != nil {
					return err
				}
				u := &unstructured.Unstructured{Object: m}
				if err := visitor.Visit(ctx, u, handler, visitDescendants, level); err != nil {
					return errors.Wrapf(err, "configmap %s visit pod %s",
						kubernetes.PrintObject(configMap), kubernetes.PrintObject(pod))
				}
				return handler.AddEdge(ctx, object, u, level)
			})
		}
		return nil
	})

	return g.Wait()
}
