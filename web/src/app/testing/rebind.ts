/*
 * Copyright (c) 2026 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import { ComponentFixture } from '@angular/core/testing';

/**
 * Applies a change to a fixture's host component and re-renders it.
 *
 * Angular 22 made `OnPush` the implicit default and stopped refreshing a
 * fixture's view on `detectChanges()` unless something marked it dirty. A
 * fixture's view is dirty when it is first created, so the initial
 * `detectChanges()` still renders — but assigning a plain field on
 * `fixture.componentInstance` afterwards marks nothing, so the second
 * `detectChanges()` is a no-op and the DOM silently keeps its first value.
 *
 * Marking the view for check restores the pre-22 "re-evaluate my template"
 * behaviour these specs were written against. This is a test-harness concern
 * only: every component in the app declares its strategy explicitly, so the
 * application's change detection is unchanged.
 */
export function rebind<T>(fixture: ComponentFixture<T>, mutate: () => void) {
  mutate();
  fixture.changeDetectorRef.markForCheck();
  fixture.detectChanges();
}
