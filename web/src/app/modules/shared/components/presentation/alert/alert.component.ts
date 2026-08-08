import {
  ChangeDetectionStrategy,
  Component,
  Input,
  OnInit,
} from '@angular/core';

import '@cds/core/alert/register.js';
import { Alert } from '../../../models/content';
import type { AlertGroupTypes, AlertStatusTypes } from '@cds/core/alert';

// Go's AlertStatus vocabulary mapped onto CDS's -- 'error' is 'danger' there.
const alertLookup: { [key: string]: AlertStatusTypes } = {
  error: 'danger',
  warning: 'warning',
  info: 'info',
  success: 'success',
};

@Component({
  selector: 'app-alert',
  templateUrl: './alert.component.html',
  styleUrls: ['./alert.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: false,
})
export class AlertComponent implements OnInit {
  @Input() alert: Alert;
  message = '';
  status: AlertStatusTypes = 'danger';
  type: AlertGroupTypes = 'default';
  closable = false;
  buttonGroup = null;
  showAlert = false;

  constructor() {}

  ngOnInit(): void {
    if (this.alert) {
      this.type = this.alert.type as typeof this.type;
      this.message = this.alert.message;
      this.status = alertLookup[this.alert.status] || alertLookup.error;
      this.closable = this.alert.closable;
      this.buttonGroup = this.alert.buttonGroup;
      this.showAlert = true;
    }
  }

  close(): void {
    this.showAlert = false;
  }
}
