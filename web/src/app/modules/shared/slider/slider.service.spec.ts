// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import { TestBed, waitForAsync } from '@angular/core/testing';
import { EditorComponent } from '../components/smart/editor/editor.component';
import { SliderService } from './slider.service';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';

describe('SliderService', () => {
  let service: SliderService;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      imports: [OverlayscrollbarsModule],
      declarations: [EditorComponent],
      providers: [],
    });
    service = TestBed.inject(SliderService);
  }));

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('set value', waitForAsync(() => {
    service.setHeight(100);
    service.setHeight$.subscribe(current => expect(current).toEqual(100));
  }));

  it('reset to default', waitForAsync(() => {
    service.resetDefault();
    service.setHeight$.subscribe(current => expect(current).toEqual(36));
  }));
});
