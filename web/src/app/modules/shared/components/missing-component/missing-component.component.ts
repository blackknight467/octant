import { Component, Input, ChangeDetectionStrategy } from '@angular/core';

@Component({
  selector: 'app-missing-component',
  templateUrl: './missing-component.component.html',
  styleUrls: ['./missing-component.component.sass'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class MissingComponentComponent {
  @Input() name: string;

  constructor() {}
}
