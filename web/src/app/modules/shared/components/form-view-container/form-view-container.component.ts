import {
  Component,
  Input,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';
import { ActionField, ActionForm } from '../../models/content';
import {
  UntypedFormArray,
  UntypedFormControl,
  UntypedFormGroup,
} from '@angular/forms';
import trackByIndex from 'src/app/util/trackBy/trackByIndex';

import '@cds/core/checkbox/register.js';
import '@cds/core/input/register.js';
import '@cds/core/textarea/register.js';
import '@cds/core/input/register.js';
import '@cds/core/radio/register.js';
import '@cds/core/select/register.js';
import { Choice } from '../../models/form-helper';

@Component({
  selector: 'app-form-view-container',
  templateUrl: './form-view-container.component.html',
  styleUrls: ['./form-view-container.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class FormViewContainerComponent implements OnInit {
  @Input()
  form: ActionForm;
  @Input()
  formGroupContainer: UntypedFormGroup;

  formArray: UntypedFormArray;

  trackByFn = trackByIndex;

  ngOnInit(): void {}

  onCheck(event, field: string) {
    this.formArray = this.formGroupContainer.get(field) as UntypedFormArray;
    if (event.target.checked) {
      this.formArray.push(new UntypedFormControl(event.target.value));
    } else {
      this.formArray.controls.forEach(
        (fc: UntypedFormControl, index: number) => {
          if (fc.value === event.target.value) {
            this.formArray.removeAt(index);
          }
        }
      );
    }
  }

  onSelect(event, field: string): void {
    this.formArray = this.formGroupContainer.get(field) as UntypedFormArray;
    this.formArray.clear();

    const selectedOptions = (event.target as HTMLSelectElement).selectedOptions;
    Array.from(selectedOptions).forEach(options => {
      this.formArray.push(new UntypedFormControl(options.value));
    });
  }

  fieldChoices(field: ActionField) {
    return field.config.configuration.choices as Choice[];
  }

  isInvalid(fieldName: string): boolean {
    const field = this.formGroupContainer.get(fieldName);
    return field.invalid && (field.dirty || field.touched);
  }
}
