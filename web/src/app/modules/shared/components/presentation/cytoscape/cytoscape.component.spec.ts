// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';

import { CytoscapeComponent } from './cytoscape.component';
import { waitFor } from 'src/app/testing/wait-for';

describe('CytoscapeComponent', () => {
  let component: CytoscapeComponent;
  let fixture: ComponentFixture<CytoscapeComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      declarations: [CytoscapeComponent],
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CytoscapeComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
    expect(component.nodes.length).toEqual(0);
  });

  it('should select first node', async () => {
    component.elements = {
      nodes: [
        {
          data: {
            id: '16428c94-a848-47d5-b1e3-c8245b57959b',
            label1: 'metadata-proxy-v0.1',
            label2: 'apps/v1 DaemonSet',
            weight: 100,
            status: 'ok',
            colorCode: '#60b515',
          },
        },
      ],
      edges: [],
    };

    component.selectedNodeId = '16428c94-a848-47d5-b1e3-c8245b57959b';
    component.render();
    fixture.detectChanges();

    // Selection is applied from cytoscape's one-shot 'render' event, which fires
    // after the instance already reports its nodes — waiting only for nodes to
    // exist would sample the graph before anything is selected.
    await waitFor(
      () => component.nodes().length > 0 && component.nodes()[0].selected(),
      'cytoscape to render and select the node'
    );

    expect(component.nodes().length).toEqual(1);
    const node = component.nodes()[0];

    expect(node).not.toBeNull();
    expect(node.id()).toEqual('16428c94-a848-47d5-b1e3-c8245b57959b');
    expect(node.isNode()).toBeTrue();
    expect(node.selected()).toBeTrue();
  });
});
