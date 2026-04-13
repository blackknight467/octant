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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/octant/internal/gvk"
	"github.com/vmware-tanzu/octant/internal/queryer"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
)

// Deployment is a typed visitor for deployments.
type Deployment struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*Deployment)(nil)

// NewDeployment creates an instance of Deployment.
func NewDeployment(q queryer.Queryer) *Deployment {
	return &Deployment{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (Deployment) Supports() schema.GroupVersionKind {
	return gvk.Deployment
}

// Visit visits a deployment. It looks for associated HPAs.
func (d *Deployment) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitDeployment")
	defer span.End()

	deployment := &appsv1.Deployment{}
	if err := kubernetes.FromUnstructured(object, deployment); err != nil {
		return err
	}
	level = handler.SetLevel(deployment.Kind, level)

	var g errgroup.Group

	g.Go(func() error {
		hpas, err := d.queryer.HPAsForObject(ctx, deployment.Namespace, "apps/v1", "Deployment", deployment.Name)
		if err != nil {
			return errors.Wrapf(err, "deployment %s visit HPAs", kubernetes.PrintObject(deployment))
		}

		for i := range hpas {
			hpa := hpas[i]
			g.Go(func() error {
				m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(hpa)
				if err != nil {
					return err
				}
				u := &unstructured.Unstructured{Object: m}
				if err := visitor.Visit(ctx, u, handler, visitDescendants, level); err != nil {
					return err
				}
				return handler.AddEdge(ctx, object, u, level)
			})
		}

		return nil
	})

	return g.Wait()
}