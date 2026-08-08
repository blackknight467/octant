// Copyright (c) 2019 the Octant contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//

import {
  ChangeDetectorRef,
  Component,
  ElementRef,
  isDevMode,
  OnDestroy,
  OnInit,
  Renderer2,
  ViewChild,
  ViewEncapsulation,
  ChangeDetectionStrategy,
} from '@angular/core';
import {
  Node,
  ResourceViewerView,
} from 'src/app/modules/shared/models/content';
import cytoscape, { ElementsDefinition } from 'cytoscape';
import { AbstractViewComponent } from '../../abstract-view/abstract-view.component';
import { ELEMENTS_STYLE, ELEMENTS_STYLE_DARK } from './octant.style';
import { Router } from '@angular/router';
import { ThemeService } from '../../../services/theme/theme.service';
import { Subscription } from 'rxjs';
import { ResizeEvent } from 'angular-resizable-element';

const statusColorCodes = {
  ok: '#60b515',
  warning: '#f57600',
  error: '#e12200',
};

const edgeColorCode = '#003d79';

const defaultZoom = {
  min: 0.075,
  max: 4.0,
};

// Kinds that can be toggled off to declutter the graph.
// Secrets are off by default because shared secrets can create an unusable graph.
const FILTERABLE_KINDS = ['Secret', 'ConfigMap', 'ServiceAccount'];
const DEFAULT_HIDDEN_KINDS = new Set(['Secret']);

const MIN_DEPTH = 1;
const MAX_DEPTH_LIMIT = 10;

@Component({
  selector: 'app-view-resource-viewer',
  templateUrl: './resource-viewer.component.html',
  styleUrls: ['./resource-viewer.component.scss'],
  encapsulation: ViewEncapsulation.None,
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: false,
})
export class ResourceViewerComponent
  extends AbstractViewComponent<ResourceViewerView>
  implements OnInit, OnDestroy
{
  selectedNodeId: string;
  private subscriptionTheme: Subscription;
  resizeEdges = { left: true, right: true };
  startPosition: number;

  // Filter state
  filterableKinds = FILTERABLE_KINDS;
  hiddenKinds = new Set(DEFAULT_HIDDEN_KINDS);
  maxDepth: number | null = null;
  readonly minDepth = MIN_DEPTH;
  readonly maxDepthLimit = MAX_DEPTH_LIMIT;

  @ViewChild('resourceViewer')
  resourceViewer: ElementRef;

  @ViewChild('viewContainer')
  viewContainer: ElementRef;

  @ViewChild('statusContainer')
  statusContainer: ElementRef;

  layout = {
    name: 'dagre',
    padding: 0,
    nodeSep: 50,
    rankSep: 150,
    rankDir: 'TB',
    directed: true,
    animate: false,
  };

  zoom = defaultZoom;

  style: cytoscape.StylesheetJson = ELEMENTS_STYLE;
  graphData: ElementsDefinition;

  constructor(
    private renderer: Renderer2,
    private router: Router,
    private themeService: ThemeService,
    private cdr: ChangeDetectorRef
  ) {
    super();
  }

  ngOnInit(): void {
    this.subscriptionTheme = this.themeService.themeType.subscribe(() => {
      this.style = this.themeService.isLightThemeEnabled()
        ? ELEMENTS_STYLE
        : ELEMENTS_STYLE_DARK;
      this.cdr.detectChanges();
    });
  }

  ngOnDestroy(): void {
    this.subscriptionTheme?.unsubscribe();
  }

  update() {
    const nodes: Node[] = this.v.config.nodes;
    if (nodes && Object.keys(nodes).length > 0) {
      const selection = this.v.config?.selected
        ? this.v.config.selected
        : Object.keys(nodes)[0];

      this.graphData = this.applyFilters();
      this.selectNode(selection);

      if (isDevMode()) {
        console.log(
          'Resource view data:',
          JSON.stringify((this.view as ResourceViewerView).config)
        );
      }
    }
  }

  // Recompute filtered graph and trigger re-render
  applyFilters(): ElementsDefinition {
    const rawNodes = this.v?.config?.nodes;
    // The backend marshals edges with `omitempty`, so a resource with no
    // relationships arrives with no edges key at all. That is a graph of
    // isolated nodes, not an empty graph — treating it as empty blanked the
    // viewer for any object that has nothing pointing at it.
    const rawEdges = this.v?.config?.edges ?? {};
    if (!rawNodes) {
      return { nodes: [], edges: [] };
    }

    // Step 1: determine which node IDs to show (filter by kind)
    const visibleNodeIds = new Set<string>(
      Object.entries(rawNodes)
        .filter(([, node]) => !this.hiddenKinds.has((node as any).kind))
        .map(([id]) => id)
    );

    // Step 2: if maxDepth is set, further restrict to nodes within depth of selected
    let allowedNodeIds: Set<string>;
    if (this.maxDepth !== null && this.selectedNodeId) {
      allowedNodeIds = this.nodesWithinDepth(
        this.selectedNodeId,
        rawEdges,
        visibleNodeIds,
        this.maxDepth
      );
    } else {
      allowedNodeIds = visibleNodeIds;
    }

    // Step 3: build cytoscape elements
    const nodeElements = Object.entries(rawNodes)
      .filter(([id]) => allowedNodeIds.has(id))
      .map(([id, details]: [string, any]) => {
        const colorCode =
          statusColorCodes[details.status] || statusColorCodes.error;
        return {
          data: {
            id,
            label1: this.getLabel(details.name, 20),
            label2: this.getLabel(`${details.apiVersion} ${details.kind}`, 36),
            weight: 100,
            status: details.status,
            colorCode,
          },
        };
      });

    const edgeElements: any[] = [];
    Object.entries(rawEdges).forEach(([parent, maps]: [string, any[]]) => {
      if (!allowedNodeIds.has(parent)) return;
      maps.forEach(edge => {
        if (!allowedNodeIds.has(edge.node)) return;
        edgeElements.push({
          data: {
            source: parent,
            target: edge.node,
            colorCode: edgeColorCode,
            strength: 10,
          },
        });
      });
    });

    return { nodes: nodeElements, edges: edgeElements };
  }

  // BFS from rootId through visible nodes, returning all within maxDepth hops
  private nodesWithinDepth(
    rootId: string,
    rawEdges: { [key: string]: any[] },
    visibleNodeIds: Set<string>,
    maxDepth: number
  ): Set<string> {
    // Build undirected adjacency for traversal
    const adj = new Map<string, Set<string>>();
    const addAdj = (a: string, b: string) => {
      if (!adj.has(a)) adj.set(a, new Set());
      if (!adj.has(b)) adj.set(b, new Set());
      adj.get(a).add(b);
      adj.get(b).add(a);
    };

    Object.entries(rawEdges).forEach(([parent, maps]: [string, any[]]) => {
      maps.forEach(edge => addAdj(parent, edge.node));
    });

    const visited = new Set<string>();
    const queue: Array<{ id: string; depth: number }> = [
      { id: rootId, depth: 0 },
    ];
    visited.add(rootId);

    while (queue.length > 0) {
      const { id, depth } = queue.shift();
      if (!visibleNodeIds.has(id)) continue;
      if (depth >= maxDepth) continue;

      (adj.get(id) || new Set()).forEach(neighbor => {
        if (!visited.has(neighbor) && visibleNodeIds.has(neighbor)) {
          visited.add(neighbor);
          queue.push({ id: neighbor, depth: depth + 1 });
        }
      });
    }

    // Keep only nodes that are both visible and within depth
    const result = new Set<string>();
    visited.forEach(id => {
      if (visibleNodeIds.has(id)) result.add(id);
    });
    return result;
  }

  toggleKind(kind: string): void {
    if (this.hiddenKinds.has(kind)) {
      this.hiddenKinds.delete(kind);
    } else {
      this.hiddenKinds.add(kind);
    }
    this.hiddenKinds = new Set(this.hiddenKinds); // trigger change detection
    this.graphData = this.applyFilters();
  }

  isKindVisible(kind: string): boolean {
    return !this.hiddenKinds.has(kind);
  }

  incrementDepth(): void {
    if (this.maxDepth === null) {
      this.maxDepth = 1;
    } else if (this.maxDepth < MAX_DEPTH_LIMIT) {
      this.maxDepth++;
    }
    this.graphData = this.applyFilters();
  }

  decrementDepth(): void {
    if (this.maxDepth !== null && this.maxDepth > MIN_DEPTH) {
      this.maxDepth--;
    } else {
      this.maxDepth = null; // below min → remove cap
    }
    this.graphData = this.applyFilters();
  }

  clearDepth(): void {
    this.maxDepth = null;
    this.graphData = this.applyFilters();
  }

  nodeChange(event) {
    this.selectNode(event.id);
    if (this.maxDepth !== null) {
      this.graphData = this.applyFilters(); // recompute depth from new selection
    }
  }

  selectNode(id: string) {
    this.selectedNodeId = id;
  }

  selectedNode(): Node {
    return this.v?.config?.nodes[this.selectedNodeId];
  }

  openNode(event) {
    const node = this.v.config.nodes[event.id];
    if (node && node.path) {
      this.router.navigateByUrl(node.path.config.ref);
    }
  }

  getLabel(label: string, length: number): string {
    return label.length > length ? label.substring(0, length) + '...' : label;
  }

  resizeCursors() {
    return {
      topLeft: 'nw-resize',
      topRight: 'ne-resize',
      bottomLeft: 'sw-resize',
      bottomRight: 'se-resize',
      leftOrRight: 'ew-resize',
      topOrBottom: 'ns-resize',
    };
  }

  resizeEnd() {
    this.zoom = Object.assign({}, defaultZoom); // update without layout recalc
  }

  resizeStart() {
    this.startPosition = this.viewContainer.nativeElement.offsetWidth;
  }

  updateSliderPosition(event: ResizeEvent) {
    const parentWidth = this.resourceViewer.nativeElement.offsetWidth - 4;
    const sliderOffset = event.edges.left as number;
    const leftSize = Math.max(
      30,
      Math.min(80, (100 * (this.startPosition + sliderOffset)) / parentWidth)
    );

    this.renderer.setStyle(
      this.viewContainer.nativeElement,
      'width',
      `${leftSize}%`
    );
    this.renderer.setStyle(
      this.statusContainer.nativeElement,
      'width',
      `${100 - leftSize}%`
    );
  }
}
