// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import {
  Component,
  Input,
  ViewEncapsulation,
  ChangeDetectionStrategy,
} from '@angular/core';
import { Node } from 'src/app/modules/shared/models/content';

@Component({
  selector: 'app-view-object-status',
  templateUrl: './object-status.component.html',
  styleUrls: ['./object-status.component.scss'],
  encapsulation: ViewEncapsulation.Emulated,
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class ObjectStatusComponent {
  @Input() node: Node;

  constructor() {}

  indicatorClass() {
    if (!this.node) {
      return ['progress', 'top', 'success'];
    }

    return [
      'progress',
      'top',
      this.node.status === 'ok' ? 'success' : 'danger',
    ];
  }

  detailsTrackBy(index, _) {
    return index;
  }

  propertiesTrackBy(index, _) {
    return index;
  }
}
