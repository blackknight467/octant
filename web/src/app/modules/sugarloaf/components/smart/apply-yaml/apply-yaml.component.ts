/*
Copyright (c) 2020 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

import {
  Component,
  HostListener,
  OnInit,
  ChangeDetectionStrategy,
} from '@angular/core';
import '@cds/core/modal/register';
import { EditorView } from 'src/app/modules/shared/models/content';

@Component({
  selector: 'app-apply-yaml',
  templateUrl: './apply-yaml.component.html',
  styleUrls: ['./apply-yaml.component.scss'],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class ApplyYAMLComponent implements OnInit {
  editorView: EditorView = {
    config: {
      value: '',
      language: 'yaml',
      readOnly: false,
      metadata: null,
      submitAction: 'action.octant.dev/apply',
      submitLabel: 'Apply',
    },
    metadata: {
      type: 'editor',
      title: [],
    },
  };

  constructor() {}

  ngOnInit() {}

  @HostListener('window:keydown', ['$event'])
  keyEvent(event: KeyboardEvent) {
    if (event.ctrlKey && event.key === 'y') {
      event.preventDefault();
      event.cancelBubble = true;
      this.toggleModal();
    }
  }

  toggleModal() {
    const yamlModal = document.getElementById('apply-yaml-modal');
    yamlModal.hidden = !yamlModal.hidden;
  }
}
