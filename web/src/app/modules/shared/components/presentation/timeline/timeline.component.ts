// Copyright (c) 2021 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import { ChangeDetectionStrategy, Component, OnInit } from '@angular/core';
import { ClrTimelineLayout, ClrTimelineStepState } from '@clr/angular';
import { AbstractViewComponent } from '../../abstract-view/abstract-view.component';
import { TimelineStep, TimelineView } from '../../../models/content';

@Component({
  selector: 'app-view-timeline',
  templateUrl: './timeline.component.html',
  styleUrls: ['./timeline.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: false,
})
export class TimelineComponent
  extends AbstractViewComponent<TimelineView>
  implements OnInit
{
  vertical: boolean;

  // Clarity types these as enums rather than string unions, and the backend
  // sends the enum's own string values, so map at the boundary.
  get layout(): ClrTimelineLayout {
    return this.vertical
      ? ClrTimelineLayout.VERTICAL
      : ClrTimelineLayout.HORIZONTAL;
  }

  stepState(step: TimelineStep): ClrTimelineStepState {
    return step.state as ClrTimelineStepState;
  }
  steps: TimelineStep[];
  constructor() {
    super();
  }
  update() {
    const view = this.v;
    this.vertical = view.config.vertical;
    this.steps = view.config.steps;
  }
  trackByFn(index, _) {
    return index;
  }
}
