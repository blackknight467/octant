// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import { Injectable } from '@angular/core';
import { ClarityIcons } from '@cds/core/icon';

export interface IconAble {
  iconName?: string;
  iconSource?: string;
}

/**
 * Strips intrinsic width/height from a custom icon SVG so it scales to its host.
 *
 * @clr/icons used to normalise injected SVGs; Clarity 18's ClrIcon renders the
 * source verbatim, so an SVG declaring e.g. width="106px" height="126px" paints at
 * that size inside a 46x46 host and is clipped to its top-left corner. Keeping the
 * viewBox and dropping the fixed size lets the host's own dimensions win.
 */
export function scalableSvg(source: string): string {
  if (!source) {
    return source;
  }

  return source.replace(/<svg\b[^>]*>/i, openingTag =>
    openingTag.replace(/\s(width|height)\s*=\s*("[^"]*"|'[^']*')/gi, '')
  );
}

@Injectable({
  providedIn: 'root',
})
export class IconService {
  constructor() {}

  /**
   * Registers a custom icon shape so <cds-icon shape="..."> can render it.
   *
   * Clarity 18 removed the legacy @clr/icons global (ClarityIcons.has/.add);
   * shapes now go into the @cds/core registry as [name, svg] tuples.
   */
  load(item: IconAble): string {
    if (!item.iconName || item.iconName === '') {
      return '';
    }

    if (item.iconSource) {
      ClarityIcons.addIcons([item.iconName, scalableSvg(item.iconSource)]);
    }

    return item.iconName;
  }
}
