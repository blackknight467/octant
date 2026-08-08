// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import { Component, ChangeDetectionStrategy } from '@angular/core';
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';

import { TextView } from '../../../models/content';
import { TextComponent } from './text.component';
import { Status } from '../indicator/indicator.component';
import { ClarityModule } from '@clr/angular';
import { rebind } from '../../../../../testing/rebind';
@Component({
  template: '<app-view-text [view]="view"></app-view-text>',
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
class TestWrapperComponent {
  view: TextView;
}

describe('TextComponent', () => {
  describe('handle changes', () => {
    let component: TestWrapperComponent;
    let fixture: ComponentFixture<TestWrapperComponent>;

    beforeEach(waitForAsync(() => {
      TestBed.configureTestingModule({
        declarations: [TestWrapperComponent, TextComponent],
        imports: [ClarityModule],
      }).compileComponents();
    }));

    beforeEach(() => {
      fixture = TestBed.createComponent(TestWrapperComponent);
      component = fixture.componentInstance;
    });

    it('should show text', () => {
      const element: HTMLDivElement = fixture.nativeElement;
      component.view = {
        config: { value: '*text*' },
        metadata: { type: 'text', title: [], accessor: 'accessor' },
      };
      fixture.detectChanges();

      expect(element.querySelector('app-view-text div')).toBeNull();
      expect(element.querySelector('app-view-text').innerHTML).toContain(
        '*text*'
      );
    });

    describe('status', () => {
      let element: HTMLDivElement;

      beforeEach(() => {
        element = fixture.nativeElement;
        component.view = {
          config: { value: 'text' },
          metadata: { type: 'text', title: [], accessor: 'accessor' },
        };
      });
      describe('with status', () => {
        beforeEach(() => {
          component.view.config.status = Status.Ok;
        });

        it('has an indicator component', () => {
          fixture.detectChanges();
          expect(
            element.querySelector('app-view-text app-indicator')
          ).not.toBeNull();
        });
      });

      describe('without status', () => {
        beforeEach(() => {
          component.view.config.status = undefined;
          fixture.detectChanges();
        });

        it('does not have an indicator', () => {
          fixture.detectChanges();
          expect(
            element.querySelector('app-view-text app-indicator')
          ).toBeNull();
        });
      });
    });

    it('should strip clarity elements with untrusted markdown', () => {
      const element: HTMLDivElement = fixture.nativeElement;
      component.view = {
        config: {
          value:
            '<clr-icon shape="check-circle" class="is-solid is-success" title="Succeeded"></clr-icon> *text*',
          isMarkdown: true,
        },
        metadata: { type: 'text', title: [], accessor: 'accessor' },
      };
      fixture.detectChanges();

      expect(element.querySelector('app-view-text markdown')).toBeDefined();
      expect(element.querySelector('app-view-text').innerHTML).toContain(
        '<p> <em>text</em></p>'
      );
    });

    it('should show clarity elements with trusted markdown', () => {
      const element: HTMLDivElement = fixture.nativeElement;
      component.view = {
        config: {
          value:
            '<clr-icon shape="check-circle" class="is-solid is-success" title="Succeeded"></clr-icon> *text*',
          isMarkdown: true,
          trustedContent: true,
        },
        metadata: { type: 'text', title: [], accessor: 'accessor' },
      };
      fixture.detectChanges();

      expect(element.querySelector('app-view-text markdown')).toBeDefined();
      expect(element.querySelector('app-view-text').innerHTML).toContain(
        '</clr-icon> <em>text</em>'
      );
    });

    it('should show markdown text', () => {
      const element: HTMLDivElement = fixture.nativeElement;
      component.view = {
        config: { value: '*text*', isMarkdown: true },
        metadata: { type: 'text', title: [], accessor: 'accessor' },
      };
      fixture.detectChanges();

      expect(element.querySelector('app-view-text markdown')).toBeDefined();
      expect(element.querySelector('app-view-text').innerHTML).toContain(
        '<p><em>text</em></p>'
      );
    });

    it('should render new markdown text after value has changed', () => {
      component.view = {
        config: { value: '*text*', isMarkdown: true },
        metadata: { type: 'text', title: [], accessor: 'accessor' },
      };
      fixture.detectChanges();

      let element: HTMLElement = fixture.nativeElement;
      expect(element.querySelector('app-view-text div').innerHTML).toEqual(
        '<p><em>text</em></p>\n'
      );

      rebind(fixture, () => {
        component.view = {
          config: { value: '# header', isMarkdown: true },
          metadata: { type: 'text', title: [], accessor: 'accessor' },
        };
      });

      element = fixture.nativeElement;
      expect(element.querySelector('app-view-text div').innerHTML).toEqual(
        '<h1>header</h1>\n'
      );
    });
  });
});
