/*
 * Copyright (c) 2026 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

/**
 * Waits until `predicate` holds, polling until a generous cap.
 *
 * Cytoscape renders asynchronously and offers no completion hook, so specs used to
 * wait a fixed `setTimeout(..., 100)`. That is a race: on a loaded machine the graph
 * is not ready in 100ms and the spec fails intermittently. Polling keeps the
 * assertions identical while removing the timing coin flip.
 */
export function waitFor(
  predicate: () => boolean,
  description = 'condition',
  timeoutMs = 5000,
  intervalMs = 25
): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    const started = Date.now();

    const tick = () => {
      if (predicate()) {
        resolve();
        return;
      }
      if (Date.now() - started > timeoutMs) {
        reject(
          new Error(`timed out after ${timeoutMs}ms waiting for ${description}`)
        );
        return;
      }
      setTimeout(tick, intervalMs);
    };

    tick();
  });
}
