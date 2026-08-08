/*
 * Copyright (c) 2020 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import { Directive, ViewContainerRef } from '@angular/core';

@Directive({
  selector: '[appView]',
  standalone: false,
})
export class ViewHostDirective {
  constructor(public viewContainerRef: ViewContainerRef) {}
}
