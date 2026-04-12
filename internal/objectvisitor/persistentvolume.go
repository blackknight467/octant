/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package objectvisitor

import (
	"context"

	"go.opencensus.io/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/octant/internal/gvk"
	"github.com/vmware-tanzu/octant/internal/queryer"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
)

// PersistentVolume is a typed visitor for persistent volumes.
type PersistentVolume struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*PersistentVolume)(nil)

// NewPersistentVolume creates an instance of PersistentVolume.
func NewPersistentVolume(q queryer.Queryer) *PersistentVolume {
	return &PersistentVolume{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (PersistentVolume) Supports() schema.GroupVersionKind {
	return gvk.PersistentVolume
}

// Visit visits a persistent volume. It links to the bound PersistentVolumeClaim.
func (p *PersistentVolume) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitPersistentVolume")
	defer span.End()

	pv := &corev1.PersistentVolume{}
	if err := kubernetes.FromUnstructured(object, pv); err != nil {
		return err
	}
	handler.SetLevel(pv.Kind, level)

	if pv.Spec.ClaimRef == nil || pv.Spec.ClaimRef.Name == "" {
		return nil
	}

	pvc, err := p.queryer.BoundPVCForPersistentVolume(ctx, pv)
	if err != nil {
		return err
	}
	if pvc == nil {
		return nil
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pvc)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: m}

	if err := visitor.Visit(ctx, u, handler, visitDescendants, level); err != nil {
		return err
	}

	return handler.AddEdge(ctx, object, u, level)
}