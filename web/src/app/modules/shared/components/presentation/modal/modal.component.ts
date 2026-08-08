import {
  Component,
  OnDestroy,
  OnInit,
  ViewChild,
  ChangeDetectionStrategy,
} from '@angular/core';
import '@cds/core/button/register.js';
import '@cds/core/modal/register';
import { AbstractViewComponent } from '../../abstract-view/abstract-view.component';
import {
  ActionForm,
  Alert,
  ButtonView,
  ModalView,
  TitleView,
  View,
} from '../../../models/content';
import { FormComponent } from '../form/form.component';
import { ModalService } from '../../../services/modal/modal.service';
import { Subscription } from 'rxjs';
import { ActionService } from '../../../services/action/action.service';

interface Choice {
  label: string;
  value: string;
  checked: boolean;
}

@Component({
  selector: 'app-view-modal',
  templateUrl: './modal.component.html',
  styleUrls: ['./modal.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class ModalComponent
  extends AbstractViewComponent<ModalView>
  implements OnInit, OnDestroy
{
  @ViewChild('modalAppForm') modalAppForm: FormComponent;

  title: TitleView[];
  body: View;
  form: ActionForm;
  opened = false;
  // Mirrors cds-modal's own union. The backend's ModalSize is sm | lg | xl and
  // is omitempty, so an absent value has to fall back to 'default'.
  size: 'default' | 'sm' | 'lg' | 'xl';
  action: string;
  buttons: ButtonView[];
  alert: Alert;

  private modalSubscription: Subscription;

  constructor(
    private actionService: ActionService,
    private modalService: ModalService
  ) {
    super();
  }

  ngOnInit() {
    this.modalSubscription = this.modalService.isOpened.subscribe(isOpened => {
      this.opened = isOpened;
    });
  }

  ngOnDestroy(): void {
    if (this.modalSubscription) {
      this.modalSubscription.unsubscribe();
    }
  }

  update() {
    this.title = this.v.metadata.title as TitleView[];
    this.body = this.v.config.body;
    this.size = (this.v.config.size as 'sm' | 'lg' | 'xl') || 'default';
    this.form = this.v.config.form;
    this.opened = this.v.config.opened;
    this.modalService.setState(this.opened);
    this.action = this.form?.action;
    this.buttons = this.v.config.buttons;
    this.alert = this.v.config.alert;
  }

  onFormSubmit() {
    if (this.modalAppForm && this.modalAppForm.formGroup.valid) {
      this.actionService.perform({
        action: this.action,
        ...this.modalAppForm.formGroup.value,
      });
      this.opened = false;
    }
  }

  onClick(payload: {}) {
    this.actionService.perform(payload);
    this.opened = false;
  }

  toggleModal(): void {
    this.opened = !this.opened;
  }
}
