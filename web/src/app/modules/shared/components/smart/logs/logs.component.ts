// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import {
  AfterViewChecked,
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  ElementRef,
  Input,
  IterableDiffer,
  IterableDiffers,
  OnDestroy,
  OnInit,
  ViewChild,
  ViewEncapsulation,
} from '@angular/core';
import { LogEntry, LogsView } from 'src/app/modules/shared/models/content';
import {
  PodLogsService,
  PodLogsStreamer,
} from 'src/app/modules/shared/pod-logs/pod-logs.service';
import { formatDate } from '@angular/common';
import { Subscription } from 'rxjs';
import { AbstractViewComponent } from '../../abstract-view/abstract-view.component';

@Component({
  selector: 'app-logs',
  templateUrl: './logs.component.html',
  styleUrls: ['./logs.component.scss'],
  encapsulation: ViewEncapsulation.None,
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class LogsComponent
  extends AbstractViewComponent<LogsView>
  implements OnInit, OnDestroy, AfterViewChecked
{
  private logStream: PodLogsStreamer;

  private containerLogsDiffer: IterableDiffer<LogEntry>;
  @ViewChild('scrollTarget', { static: true }) scrollTarget: ElementRef;

  @Input() containerLogs: LogEntry[] = [];

  selectedContainer = '';
  selectedSince = 0;
  shouldDisplayTimestamp = false;
  shouldDisplayName = true;
  showOnlyFiltered = false;
  currentSelection = 0;
  totalSelections = 0;
  timeFormat = 'MMM d, y h:mm:ss a z';
  regexFlags = 'gi';

  private logSubscription: Subscription;

  private _filterText = '';
  private filterChanged = false;

  get filterText(): string {
    return this._filterText;
  }

  /**
   * Recomputes the match count as the filter changes rather than from
   * ngAfterContentChecked. totalSelections is template-bound, and writing it
   * from a lifecycle hook is what Angular reports as NG0100.
   */
  set filterText(value: string) {
    if (value === this._filterText) {
      return;
    }
    this._filterText = value;
    this.updateSelectedCount();
    this.filterChanged = true;
  }

  constructor(
    private podLogsService: PodLogsService,
    private iterableDiffers: IterableDiffers,
    private cdr: ChangeDetectorRef
  ) {
    super();
  }

  ngOnInit() {
    this.containerLogsDiffer = this.iterableDiffers
      .find(this.containerLogs)
      .create();
    this.startStream();
  }

  protected update() {
    if (this.v.config.containers && this.v.config.containers.length > 0) {
      this.selectedContainer = this.v.config.containers[0];
    }
  }

  onSinceChange(selectedSince: string): void {
    this.selectedSince = +selectedSince;
    this.stopStreamIfStarted();
    this.startStream();
    this.updateSelectedCount();
  }

  stopStreamIfStarted(): void {
    if (this.logStream) {
      this.containerLogs = [];
      this.logStream.close();
      this.logStream = null;
    }
  }

  onContainerChange(containerSelection: string): void {
    this.selectedContainer = containerSelection;
    this.stopStreamIfStarted();
    if (this.selectedContainer === '') {
      this.shouldDisplayName = true;
    } else {
      this.shouldDisplayName = false;
    }

    this.startStream();
  }

  toggleTimestampDisplay(): void {
    this.shouldDisplayTimestamp = !this.shouldDisplayTimestamp;
    this.updateSelectedCount();
    this.scrollToHighlight(0, 0);
  }

  toggleShowOnlyFiltered(): void {
    this.showOnlyFiltered = !this.showOnlyFiltered;
    this.scrollToHighlight(0, 0);
  }

  startStream() {
    const namespace = this.v.config.namespace;
    const pod = this.v.config.name;
    const container = this.selectedContainer;
    const since = this.selectedSince;
    if (namespace && pod) {
      this.logStream = this.podLogsService.createStream(
        namespace,
        pod,
        container,
        since
      );
      this.logSubscription = this.logStream.logEntry.subscribe(
        (entry: LogEntry) => {
          if (entry.message == null) {
            return;
          }
          this.containerLogs.push(entry);
          this.updateSelectedCount();
          this.cdr.markForCheck();
        }
      );
    }
  }

  ngAfterViewChecked() {
    this.containerLogsDiffer.diff(this.containerLogs);
    if (this.filterChanged) {
      this.filterChanged = false;
      // Needs the re-rendered highlights, so this cannot move into the setter.
      // Deferred past the current change-detection pass because it writes
      // currentSelection, which is template-bound.
      Promise.resolve().then(() => {
        this.scrollToHighlight(0, 0);
        this.cdr.markForCheck();
      });
    }
  }

  ngOnDestroy(): void {
    if (this.logStream) {
      this.logStream.close();
      this.logStream = null;
    }

    if (this.logSubscription) {
      this.logSubscription.unsubscribe();
    }
  }

  /**
   * Compiles a filter pattern, or returns null if it is not valid yet.
   *
   * The filter box is bound with ngModel, so this sees every keystroke —
   * including intermediate states like a lone "(" on the way to "(foo|bar)".
   * Those must not throw: treat an incomplete pattern as "no filter".
   */
  private safeRegex(pattern: string): RegExp | null {
    try {
      return new RegExp(pattern, this.regexFlags);
    } catch {
      return null;
    }
  }

  public highlightText(text: string) {
    if (!this.filterText) {
      return text;
    }

    const search = this.safeRegex(this.filterText);
    if (!search) {
      return text;
    }

    const matched = search.exec(text);
    if (matched === null) {
      return text;
    }

    const filter =
      matched[0] && matched[0].length > 0
        ? this.filterText
        : this.filterText + '.*$';

    const replacement = this.safeRegex(filter);
    if (!replacement) {
      return text;
    }

    return text.replace(replacement, match => {
      return '<span class="highlight">' + match + '</span>';
    });
  }

  public filterFunction(logs: LogEntry[]): LogEntry[] {
    if (this.showOnlyFiltered) {
      return logs.filter(log => {
        const hasFiltered = this.matchRegex(log);
        return hasFiltered && hasFiltered.length > 0;
      });
    }

    return logs;
  }

  onPreviousHighlight(): void {
    if (this.currentSelection > 0) {
      this.scrollToHighlight(-1);
    } else {
      this.scrollToHighlight(0, this.totalSelections - 1);
    }
  }

  onNextHighlight(): void {
    if (this.getHighlightedElement(this.currentSelection + 1)) {
      this.scrollToHighlight(1);
    } else {
      this.scrollToHighlight(0, 0);
    }
  }

  scrollToHighlight(scrollBy: number, newSelection?: number) {
    this.removeHighlightSelection();
    if (newSelection !== undefined) {
      this.currentSelection = newSelection;
    }

    if (this.getHighlightedElement(this.currentSelection + scrollBy)) {
      this.currentSelection += scrollBy;
      const nextSelection: HTMLElement = this.getHighlightedElement(
        this.currentSelection
      );
      const { clientHeight, offsetTop, scrollTop } =
        this.scrollTarget.nativeElement;
      const top = nextSelection.offsetTop - offsetTop;

      if (top > clientHeight + scrollTop || top < scrollTop) {
        nextSelection.scrollIntoView(true);
      }
      nextSelection.className = 'highlight highlight-selected';
    }
  }

  removeHighlightSelection(): HTMLElement {
    const element: HTMLElement = this.getHighlightedElement(
      this.currentSelection
    );
    if (element) {
      element.className = 'highlight';
    }
    return element;
  }

  getHighlightedElement(index: number): HTMLElement {
    return document.getElementsByClassName('highlight')[index] as HTMLElement;
  }

  matchRegex(input: LogEntry) {
    const search = this.safeRegex(this.filterText);
    if (!search) {
      return null;
    }

    let match = input.message.match(search);
    if (match) {
      return match;
    }

    if (this.shouldDisplayTimestamp && input.timestamp) {
      const timestamp = formatDate(input.timestamp, this.timeFormat, 'en-US');
      if (timestamp && timestamp.length > 0) {
        match = timestamp.match(search);
        if (match) {
          return match;
        }
      }
    }

    if (this.shouldDisplayName) {
      match = input.container.match(search);
      return match || [];
    }
    return [];
  }

  updateSelectedCount() {
    let count = 0;
    if (this.filterText.length > 0) {
      this.containerLogs.map(log => {
        count += (this.matchRegex(log) || []).length;
      });
    }
    this.totalSelections = count;
  }
}
