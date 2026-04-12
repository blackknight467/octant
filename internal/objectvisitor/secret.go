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

// Secret is a typed visitor for secrets.
type Secret struct {
	queryer queryer.Queryer
}

var _ TypedVisitor = (*Secret)(nil)

// NewSecret creates an instance of Secret.
func NewSecret(q queryer.Queryer) *Secret {
	return &Secret{queryer: q}
}

// Supports returns the gvk this typed visitor supports.
func (Secret) Supports() schema.GroupVersionKind {
	return gvk.Secret
}

// Visit visits a secret. It links to pods that reference it.
func (s *Secret) Visit(ctx context.Context, object *unstructured.Unstructured, handler ObjectHandler, visitor Visitor, visitDescendants bool, level int) error {
	ctx, span := trace.StartSpan(ctx, "visitSecret")
	defer span.End()

	secret := &corev1.Secret{}
	if err := kubernetes.FromUnstructured(object, secret); err != nil {
		return err
	}
	handler.SetLevel(secret.Kind, level)

	var g errgroup.Group

	g.Go(func() error {
		pods, err := s.queryer.PodsForSecret(ctx, secret)
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
					return errors.Wrapf(err, "secret %s visit pod %s",
						kubernetes.PrintObject(secret), kubernetes.PrintObject(pod))
				}
				return handler.AddEdge(ctx, object, u, level)
			})
		}
		return nil
	})

	return g.Wait()
}