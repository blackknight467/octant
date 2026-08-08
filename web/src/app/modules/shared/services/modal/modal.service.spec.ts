/*
 * Copyright (c) 2020 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

import { TestBed } from '@angular/core/testing';
import { ModalService } from './modal.service';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';

describe('ModalService', () => {
  beforeEach(() =>
    TestBed.configureTestingModule({
      declarations: [],
      imports: [OverlayscrollbarsModule],
      providers: [ModalService],
    })
  );

  it('should be created', () => {
    const service: ModalService = TestBed.inject(ModalService);
    expect(service).toBeTruthy();
  });
});
