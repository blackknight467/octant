import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA } from '@angular/core';
import { ClarityModule } from '@clr/angular';
import { StepperComponent } from './stepper.component';
import { UntypedFormBuilder, ReactiveFormsModule } from '@angular/forms';
import { StepperView } from '../../../models/content';
import {
  BrowserAnimationsModule,
  NoopAnimationsModule,
} from '@angular/platform-browser/animations';
import { WebsocketService } from '../../../../../data/services/websocket/websocket.service';
import { anything, deepEqual, instance, mock, verify } from 'ts-mockito';
import { ActionService } from '../../../services/action/action.service';

describe('StepperComponent', () => {
  let component: StepperComponent;
  let fixture: ComponentFixture<StepperComponent>;
  const formBuilder: UntypedFormBuilder = new UntypedFormBuilder();

  const mockActionService: ActionService = mock(ActionService);

  const action = 'action.octant.dev/test';
  const view: StepperView = {
    metadata: {
      type: 'stepper',
    },
    config: {
      action,
      steps: [
        {
          name: 'step 1',
          form: { fields: [] },
          title: 'step title',
          description: 'step description',
        },
        {
          name: 'confirmation step',
          form: { fields: [] },
          title: 'step title',
          description: 'confirmation description',
        },
      ],
    },
  };

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      // StepperComponent is not standalone, so it needs declaring here. Without
      // it the template compiles against an empty directive scope: clrStepper,
      // clrStepButton and formGroup all silently fail to attach, the buttons
      // stay plain type="submit" controls, and clicking one does a native form
      // submit that reloads the karma page.
      declarations: [StepperComponent],
      imports: [
        ClarityModule,
        ReactiveFormsModule,
        BrowserAnimationsModule,
        NoopAnimationsModule,
      ],
      // The step bodies render app-form-view-container, which is out of scope
      // for this spec; it only exercises stepping through and submitting.
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        { provide: UntypedFormBuilder, useValue: formBuilder },
        {
          provide: ActionService,
          useValue: instance(mockActionService),
        },
      ],
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(StepperComponent);
    component = fixture.componentInstance;

    component.view = view;
  });

  it('should submit form after completing each step', async () => {
    fixture.detectChanges();
    await fixture.whenStable();

    // Clarity only renders the expanded panel's body, and it expands the first
    // panel from a lifecycle hook after the initial render, so the step's
    // button does not exist until a further round of change detection.
    const render = async () => {
      fixture.changeDetectorRef.markForCheck();
      fixture.detectChanges();
      await fixture.whenStable();
    };
    const click = async (selector: string) => {
      await render();
      const button: HTMLButtonElement =
        fixture.debugElement.nativeElement.querySelector(selector);
      expect(button)
        .withContext(`expected ${selector} to be rendered`)
        .not.toBeNull();
      button.click();
      await render();
    };

    await render();
    console.log(
      'ZZ DOM:',
      fixture.debugElement.nativeElement.innerHTML.slice(0, 2000)
    );
    await click('.next');
    await click('.submit');

    verify(
      mockActionService.perform(
        deepEqual({ action, 'step 1': {}, 'confirmation step': {} })
      )
    ).once();
  });
});
