// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import { HttpClient } from '@angular/common/http';
import { HttpTestingController } from '@angular/common/http/testing';
import { DebugElement } from '@angular/core';
import { waitForAsync, ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import map from 'lodash/map';
import range from 'lodash/range';
import uniqueId from 'lodash/uniqueId';
import {
  LogEntry,
  LogsView,
  Since,
} from 'src/app/modules/shared/models/content';
import { PodLogsService } from 'src/app/modules/shared/pod-logs/pod-logs.service';
import { WebsocketService } from '../../../../../data/services/websocket/websocket.service';
import { WebsocketServiceMock } from '../../../../../data/services/websocket/mock';
import { LogsComponent } from './logs.component';
import { AnsiPipe } from '../../../pipes/ansiPipe/ansi.pipe';
import { windowProvider, WindowToken } from '../../../../../window';
import { StringEscapePipe } from '../../../pipes/stringEscape/string.escape.pipe';
import { rebind } from '../../../../../testing/rebind';
import { waitFor } from '../../../../../testing/wait-for';

function createTestLogsView(
  durations: Since[],
  containers: string[]
): LogsView {
  return {
    metadata: {
      type: 'logs',
      title: [],
      accessor: 'logs',
    },
    config: {
      namespace: 'default',
      name: 'cartpod',
      containers,
      durations,
    },
  };
}

function createRandomLogEntry(): LogEntry {
  return {
    timestamp: '2019-05-06T18:59:06.554540433Z',
    message: uniqueId('message'),
    container: 'test-container',
  };
}

describe('LogsComponent <-> PodsLogsService', () => {
  let component: LogsComponent;
  let fixture: ComponentFixture<LogsComponent>;
  let service: PodLogsService;
  let httpClient: HttpClient;
  let httpTestingController: HttpTestingController;
  const defaultTestLogs = [
    {
      timestamp: '2019-05-06T18:59:06.554540433Z',
      message: 'messageA',
      container: 'test-container',
    },
    {
      timestamp: '2019-05-06T18:59:06.554540433Z',
      message: 'messageB',
      container: 'test-container',
    },
    {
      timestamp: '2019-05-06T18:59:06.554540433Z',
      message: 'messageC',
      container: 'test-container',
    },
  ];

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      declarations: [LogsComponent, AnsiPipe, StringEscapePipe],
      providers: [
        PodLogsService,
        { provide: WindowToken, useFactory: windowProvider },
        // see logs.component.spec.ts: the real WebsocketService dials karma itself
        { provide: WebsocketService, useClass: WebsocketServiceMock },
      ],
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(LogsComponent);
    component = fixture.componentInstance;
    service = TestBed.inject(PodLogsService);
    httpClient = TestBed.inject(HttpClient);
    httpTestingController = TestBed.inject(HttpTestingController);

    component.view = createTestLogsView(
      [{ label: '5 minutes', seconds: 300 }],
      ['containerA', 'containerB', 'containerC']
    );
  });

  it('should allow user to toggle displaying timestamps', () => {
    component.shouldDisplayTimestamp = true;
    component.containerLogs = defaultTestLogs;
    fixture.detectChanges();

    let logEntriesDebugElement: DebugElement[] = fixture.debugElement.queryAll(
      By.css('.container-log')
    );
    expect(logEntriesDebugElement.length).toBe(3);
    expect(logEntriesDebugElement[0].nativeElement.textContent).toMatch(
      /May \d+, 2019(.+)messageA/
    );
    expect(logEntriesDebugElement[1].nativeElement.textContent).toMatch(
      /May \d+, 2019(.+)messageB/
    );
    expect(logEntriesDebugElement[2].nativeElement.textContent).toMatch(
      /May \d+, 2019(.+)messageC/
    );

    rebind(fixture, () => {
      component.shouldDisplayTimestamp = false;
    });

    logEntriesDebugElement = fixture.debugElement.queryAll(
      By.css('.container-log')
    );
    expect(logEntriesDebugElement.length).toBe(3);
    expect(logEntriesDebugElement[0].nativeElement.textContent).toBe(
      'test-containermessageA'
    );
    expect(logEntriesDebugElement[1].nativeElement.textContent).toBe(
      'test-containermessageB'
    );
    expect(logEntriesDebugElement[2].nativeElement.textContent).toBe(
      'test-containermessageC'
    );
  });

  it('should continuously scroll to new logs if user has already scrolled to the bottom', () => {
    const numberOfEntriesRequiredToScroll = 200;
    component.containerLogs = map(
      range(numberOfEntriesRequiredToScroll),
      createRandomLogEntry
    );
    fixture.detectChanges();

    const logWrapperDebugElement: DebugElement = fixture.debugElement.query(
      By.css('.log-container')
    );
    let logWrapperNativeElement: HTMLDivElement =
      logWrapperDebugElement.nativeElement;
    expect(logWrapperNativeElement.scrollHeight).toEqual(
      logWrapperNativeElement.clientHeight
    );
    expect(logWrapperNativeElement.scrollTop).toEqual(0);

    const logWrapperHeight = logWrapperNativeElement.clientHeight;
    logWrapperNativeElement.scrollTop = logWrapperNativeElement.scrollHeight;

    expect(logWrapperNativeElement.scrollTop).toEqual(
      logWrapperNativeElement.scrollHeight - logWrapperHeight
    );

    const newContainerLogs: LogEntry[] = map(
      range(numberOfEntriesRequiredToScroll),
      createRandomLogEntry
    );
    rebind(fixture, () => {
      component.containerLogs.push(...newContainerLogs);
      logWrapperNativeElement.dispatchEvent(new Event('scroll'));
    });

    logWrapperNativeElement = fixture.debugElement.query(
      By.css('.log-container')
    ).nativeElement;
    expect(logWrapperNativeElement.scrollTop).toEqual(0);
  });

  it('should keep scroll position even if new logs are coming in and user is not at bottom', async () => {
    const numberOfEntriesRequiredToScroll = 200;
    component.containerLogs = map(
      range(numberOfEntriesRequiredToScroll),
      createRandomLogEntry
    );
    fixture.detectChanges();

    const logWrapperDebugElement: DebugElement = fixture.debugElement.query(
      By.css('.container-logs-bg')
    );
    let logWrapperNativeElement: HTMLDivElement =
      logWrapperDebugElement.nativeElement;
    await fixture.whenStable();
    // The scroll container is sized by overlayscrollbars, which initialises
    // asynchronously, so the element still measures 0x0 immediately after
    // whenStable(). Wait for real geometry rather than assert against a
    // half-laid-out element.
    await waitFor(
      () =>
        logWrapperNativeElement.scrollHeight >
        logWrapperNativeElement.clientHeight,
      'log container to become scrollable'
    );
    expect(logWrapperNativeElement.scrollHeight).toBeGreaterThan(
      logWrapperNativeElement.clientHeight
    );
    // The log list is laid out `flex-direction: column-reverse`, so the browser
    // puts the scroll origin at the visual bottom: resting on the newest entry
    // is scrollTop === 0, and scrolling *up* into history gives negative
    // offsets (positive values clamp back to 0). This spec was written against
    // Chrome's older behaviour, which reported the same positions as positive
    // offsets. Only the reported sign changed — the on-screen behaviour, and
    // the CSS producing it, are unchanged.
    expect(logWrapperNativeElement.scrollTop).toBe(0);

    // scroll halfway up into the history
    const halfwayScrollMark = -Math.floor(
      logWrapperNativeElement.clientHeight / 2
    );
    logWrapperNativeElement.scrollTop = halfwayScrollMark;
    logWrapperNativeElement.dispatchEvent(new Event('scroll'));
    const offsetFromTopBefore = offsetFromTop(logWrapperNativeElement);

    // add new logs
    const newContainerLogs: LogEntry[] = map(
      range(numberOfEntriesRequiredToScroll),
      createRandomLogEntry
    );
    rebind(fixture, () => {
      component.containerLogs.push(...newContainerLogs);
    });

    // check scroll is in same place
    logWrapperNativeElement = fixture.debugElement.query(
      By.css('.container-logs-bg')
    ).nativeElement;
    await fixture.whenStable();
    // Because the scroll origin sits at the bottom, appending logs below the
    // viewport necessarily shifts scrollTop — holding it fixed would mean the
    // user's view slid forward onto newer lines. What must not change is how
    // far down the history they are, which is the offset from the top.
    expect(offsetFromTop(logWrapperNativeElement)).toBe(offsetFromTopBefore);
  });

  // Distance from the start of the log history, independent of whether the
  // browser reports scroll offsets from the top or from the bottom.
  function offsetFromTop(el: HTMLElement): number {
    return el.scrollTop + el.scrollHeight - el.clientHeight;
  }

  // `.highlight-selected` is applied by scrollToHighlight(), which the component
  // defers to a microtask so it does not write template-bound state during the
  // change-detection pass that rendered the highlights. Each assertion therefore
  // has to let that microtask run first.
  it('should filter messages based on search string', async () => {
    component.view = createTestLogsView(
      [{ label: '5 minutes', seconds: 300 }],
      ['containerA', 'containerB', 'containerC']
    );

    component.shouldDisplayTimestamp = true;
    component.containerLogs = defaultTestLogs;
    component.filterText = 'message';
    fixture.detectChanges();
    await fixture.whenStable();
    VerifyElementsExist('.container-log', 3);
    VerifyElementsExist('.highlight', 3);
    VerifyElementsExist('.highlight-selected', 1);

    rebind(fixture, () => {
      component.filterText = 'messageA';
    });
    await fixture.whenStable();
    VerifyElementsExist('.container-log', 3);
    VerifyElementsExist('.highlight', 1);
    VerifyElementsExist('.highlight-selected', 1);

    rebind(fixture, () => {
      component.showOnlyFiltered = true;
    });
    await fixture.whenStable();
    VerifyElementsExist('.container-log', 1);
    VerifyElementsExist('.highlight', 1);
    VerifyElementsExist('.highlight-selected', 1);

    rebind(fixture, () => {
      component.filterText = '';
    });
    await fixture.whenStable();
    VerifyElementsExist('.container-log', 3);
    VerifyElementsExist('.highlight', 0);
    VerifyElementsExist('.highlight-selected', 0);
  });

  afterEach(() => {
    httpTestingController.verify();
  });

  function VerifyElementsExist(selector: string, noItems: number) {
    const element: DebugElement[] = fixture.debugElement.queryAll(
      By.css(selector)
    );
    expect(element.length).toBe(noItems);
  }
});
