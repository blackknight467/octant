// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import {
  Component,
  Input,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';
import { ActionForm } from '../../../models/content';
import { UntypedFormBuilder, UntypedFormGroup } from '@angular/forms';
import { FormHelper } from '../../../models/form-helper';

@Component({
  selector: 'app-form',
  templateUrl: './form.component.html',
  styleUrls: ['./form.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class FormComponent implements OnInit {
  @Input()
  form: ActionForm;

  formGroup: UntypedFormGroup;

  constructor(private formBuilder: UntypedFormBuilder) {}

  ngOnInit() {
    const formHelper = new FormHelper();
    this.formGroup = formHelper.createFromGroup(this.form, this.formBuilder);
  }
}
