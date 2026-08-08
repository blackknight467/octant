// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';
import { ResourceViewerComponent } from './resource-viewer.component';
import { SharedModule } from '../../../shared.module';
import { ResourceViewerView } from '../../../models/content';
import { DebugElement } from '@angular/core';
import { By } from '@angular/platform-browser';
import { OverlayscrollbarsModule } from 'overlayscrollbars-ngx';
import { waitFor } from 'src/app/testing/wait-for';

describe('ResourceViewerComponent', () => {
  let component: ResourceViewerComponent;
  let fixture: ComponentFixture<ResourceViewerComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      declarations: [],
      imports: [SharedModule, OverlayscrollbarsModule],
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ResourceViewerComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should show node labels', async () => {
    component.view = {
      config: {
        nodes: {
          '16428c94-a848-47d5-b1e3-c8245b57959b': {
            name: 'metadata-proxy-v0.1',
            apiVersion: 'apps/v1',
            kind: 'DaemonSet',
            status: 'ok',
            details: [
              {
                metadata: { type: 'text' },
                config: { value: 'Daemon Set is OK' },
              },
            ],
            path: {
              metadata: {
                type: 'link',
                title: [{ metadata: { type: 'text' }, config: { value: '' } }],
              },
              config: {
                value: 'metadata-proxy-v0.1',
                ref: '/overview/namespace/kube-system/workloads/daemon-sets/metadata-proxy-v0.1',
              },
            },
            hasChildren: false,
          },
        },
      },
    } as unknown as ResourceViewerView;

    fixture.detectChanges();

    await waitFor(
      () => fixture.debugElement.queryAll(By.css('.label1')).length > 0,
      'cytoscape to render node labels'
    );

    const header: DebugElement[] = fixture.debugElement.queryAll(
      By.css('.label1')
    );
    expect(header.length).toEqual(1);
  });
});
