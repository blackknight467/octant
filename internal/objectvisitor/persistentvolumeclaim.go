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

// PersistentVolumeClaim is a typed visitor for persistent volume claims.
type PersistentVolumeClaim struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*PersistentVolumeClaim)(nil)

// NewPersistentVolumeClaim creates an instance of PersistentVolumeClaim.
func NewPersistentVolumeClaim(q queryer.Queryer) *PersistentVolumeClaim {
	return &PersistentVolumeClaim{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (PersistentVolumeClaim) Supports() schema.GroupVersionKind {
	return gvk.PersistentVolumeClaim
}

// Visit visits a persistent volume claim. It links to the bound PersistentVolume and to pods that mount it.
func (p *PersistentVolumeClaim) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitPersistentVolumeClaim")
	defer span.End()

	pvc := &corev1.PersistentVolumeClaim{}
	if err := kubernetes.FromUnstructured(object, pvc); err != nil {
		return err
	}
	handler.SetLevel(pvc.Kind, level)

	var g errgroup.Group

	// Link to bound PersistentVolume
	g.Go(func() error {
		if pvc.Spec.VolumeName == "" {
			return nil
		}
		pv, err := p.queryer.BoundPersistentVolumeForPVC(ctx, pvc)
		if err != nil {
			return err
		}
		if pv == nil {
			return nil
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pv)
		if err != nil {
			return err
		}
		u := &unstructured.Unstructured{Object: m}
		return handler.AddEdge(ctx, object, u, level)
	})

	// Link to pods that mount this PVC
	g.Go(func() error {
		pods, err := p.queryer.PodsForPVC(ctx, pvc)
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
					return errors.Wrapf(err, "pvc %s visit pod %s",
						kubernetes.PrintObject(pvc), kubernetes.PrintObject(pod))
				}
				return handler.AddEdge(ctx, object, u, level)
			})
		}
		return nil
	})

	return g.Wait()
}
