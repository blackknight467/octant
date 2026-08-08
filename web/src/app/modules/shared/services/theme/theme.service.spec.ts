// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import { inject, TestBed } from '@angular/core/testing';
import { ThemeService } from './theme.service';
import { DOCUMENT } from '@angular/common';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';

describe('ThemeService', () => {
  let service: ThemeService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [OverlayscrollbarsModule],
      declarations: [],
      providers: [ThemeService, Document],
    });

    service = TestBed.inject(ThemeService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should apply the light theme via the cds-theme attribute', inject(
    [DOCUMENT],
    (document: Document) => {
      service.applyTheme('light');

      expect(document.body.getAttribute('cds-theme')).toBe('light');
      expect(document.body.classList.contains('light')).toBe(true);
      expect(document.body.classList.contains('dark')).toBe(false);
    }
  ));

  it('should apply the dark theme via the cds-theme attribute', inject(
    [DOCUMENT],
    (document: Document) => {
      service.applyTheme('dark');

      expect(document.body.getAttribute('cds-theme')).toBe('dark');
      expect(document.body.classList.contains('dark')).toBe(true);
      expect(document.body.classList.contains('light')).toBe(false);
    }
  ));
});
