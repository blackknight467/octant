// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import {
  AfterViewChecked,
  Component,
  ElementRef,
  Input,
  ChangeDetectionStrategy,
} from '@angular/core';
import trackByIdentity from 'src/app/util/trackBy/trackByIdentity';
import {
  ExpressionSelectorView,
  LabelSelectorView,
} from 'src/app/modules/shared/models/content';

@Component({
  selector: 'app-overflow-selectors',
  templateUrl: './overflow-selectors.component.html',
  styleUrls: ['./overflow-selectors.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class OverflowSelectorsComponent implements AfterViewChecked {
  // metadata.type is a plain string on both members, so @switch cannot narrow
  // the union. These make the branch's intent explicit at the call site.
  asLabel(selector: ExpressionSelectorView | LabelSelectorView) {
    return selector as LabelSelectorView;
  }

  asExpression(selector: ExpressionSelectorView | LabelSelectorView) {
    return selector as ExpressionSelectorView;
  }

  @Input() set selectors(
    selectors: Array<LabelSelectorView | ExpressionSelectorView>
  ) {
    this.selectorsList = selectors;
    this.updateSelectors();
  }

  get selectors(): Array<LabelSelectorView | ExpressionSelectorView> {
    return this.selectorsList;
  }

  constructor(private rootElement: ElementRef) {}
  @Input() numberShownSelectors = 2;

  private selectorsList: Array<LabelSelectorView | ExpressionSelectorView>;
  showSelectors: Array<LabelSelectorView | ExpressionSelectorView>;
  overflowSelectors: Array<LabelSelectorView | ExpressionSelectorView>;
  trackByIdentity = trackByIdentity;
  componentWidth = 0;

  private updateSelectors() {
    if (this.numberShownSelectors <= this.selectorsList.length) {
      this.showSelectors = this.selectorsList.slice(
        0,
        this.numberShownSelectors
      );
      this.overflowSelectors = this.selectorsList.slice(
        this.numberShownSelectors
      );
    } else {
      this.showSelectors = this.selectorsList;
    }
  }

  ngAfterViewChecked(): void {
    if (this.componentWidth !== this.rootElement.nativeElement.clientWidth) {
      this.numberShownSelectors =
        this.rootElement.nativeElement.clientWidth > 150 ? 2 : 1;
      this.updateSelectors();
      this.componentWidth = this.rootElement.nativeElement.clientWidth;
    }
  }
}
