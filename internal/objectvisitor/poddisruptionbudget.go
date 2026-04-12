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
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/octant/internal/gvk"
	"github.com/vmware-tanzu/octant/internal/queryer"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
)

// PodDisruptionBudget is a typed visitor for pod disruption budgets.
type PodDisruptionBudget struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*PodDisruptionBudget)(nil)

// NewPodDisruptionBudget creates an instance of PodDisruptionBudget.
func NewPodDisruptionBudget(q queryer.Queryer) *PodDisruptionBudget {
	return &PodDisruptionBudget{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (PodDisruptionBudget) Supports() schema.GroupVersionKind {
	return gvk.PodDisruptionBudget
}

// Visit visits a pod disruption budget. It looks for associated pods.
func (p *PodDisruptionBudget) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitPodDisruptionBudget")
	defer span.End()

	pdb := &policyv1.PodDisruptionBudget{}
	if err := kubernetes.FromUnstructured(object, pdb); err != nil {
		return err
	}
	level = handler.SetLevel(pdb.Kind, level)

	var g errgroup.Group

	g.Go(func() error {
		pods, err := p.queryer.PodsForPodDisruptionBudget(ctx, pdb)
		if err != nil {
			return errors.Wrapf(err, "pod disruption budget %s visit pods", kubernetes.PrintObject(pdb))
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
					return err
				}
				return handler.AddEdge(ctx, object, u, level)
			})
		}

		return nil
	})

	return g.Wait()
}