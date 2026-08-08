// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

/**
 * Central icon registration.
 *
 * Clarity 18 removed the legacy @clr/icons global registry that used to
 * populate <clr-icon> automatically. Shapes must now be registered with the
 * @cds/core registry before they can render — an unregistered shape shows a
 * three-dot loading placeholder rather than erroring, so this failure is
 * invisible to the compiler and to unit tests.
 *
 * Whole collections are loaded rather than individual shapes because icons are
 * requested from three places that cannot all be enumerated statically:
 *   - octant's own templates (shape="...")
 *   - the Go backend, which sends icon names at runtime
 *   - Clarity's own components (datagrid filters, dropdowns, steppers), which
 *     reference shapes like filter-grid, ellipsis-vertical and check-circle
 *
 * Custom/app-specific shapes (e.g. octant-logo) go through IconService.
 */
import {
  ClarityIcons,
  loadChartIconSet,
  loadCoreIconSet,
  loadEssentialIconSet,
  loadMediaIconSet,
  loadMiniIconSet,
  loadTechnologyIconSet,
  loadTextEditIconSet,
} from '@cds/core/icon';

let registered = false;

export function registerOctantIcons(): void {
  if (registered) {
    return;
  }
  loadCoreIconSet();
  loadEssentialIconSet();
  loadMiniIconSet();
  loadTechnologyIconSet();
  loadChartIconSet();
  loadMediaIconSet();
  loadTextEditIconSet();

  // Clarity's stack view and some backend-driven statuses ask for "unknown",
  // which ships as "unknown-status" in the core set.
  const unknownStatus = ClarityIcons.registry['unknown-status'];
  if (unknownStatus) {
    ClarityIcons.addIcons(['unknown', unknownStatus]);
  }

  // @cds/core ships a `sideEffects` allowlist that covers only its register.js
  // entrypoints, so the optimizer treats these loader calls as pure and strips
  // both them and the icon data from the production bundle. Reading the registry
  // back and publishing the count creates an observable dependency the optimizer
  // cannot remove, which forces the loaders (and their shapes) to be retained.
  const shapes = Object.keys(ClarityIcons.registry);
  (window as unknown as Record<string, unknown>).__octantIconShapes =
    shapes.length;

  registered = true;
}
