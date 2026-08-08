// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
import { Inject, Injectable, Renderer2, RendererFactory2 } from '@angular/core';
import { DOCUMENT } from '@angular/common';
import { BehaviorSubject } from 'rxjs';

export type ThemeType = 'light' | 'dark';

export interface Theme {
  type: ThemeType;
}

/**
 * Dark theme
 */
export const darkTheme: Theme = {
  type: 'dark',
};

/**
 * Light theme
 */
export const lightTheme: Theme = {
  type: 'light',
};

export const defaultTheme = window.matchMedia('(prefers-color-scheme:dark)')
  .matches
  ? darkTheme
  : lightTheme;

@Injectable({
  providedIn: 'root',
})
export class ThemeService {
  public themeType: BehaviorSubject<ThemeType> = new BehaviorSubject<ThemeType>(
    defaultTheme.type
  );

  private renderer: Renderer2;

  constructor(
    @Inject(DOCUMENT) private document: Document,
    rendererFactory: RendererFactory2
  ) {
    this.renderer = rendererFactory.createRenderer(null, null);
  }

  loadTheme(): void {
    this.applyTheme(
      this.isLightThemeEnabled() ? lightTheme.type : darkTheme.type
    );
  }

  /**
   * Applies a theme. Clarity 17 ships a single stylesheet and selects the
   * palette from the cds-theme attribute, so there is no stylesheet to swap.
   * The body class is kept for component styles using :host-context(body.dark).
   */
  applyTheme(type: ThemeType): void {
    [darkTheme, lightTheme].forEach(t =>
      this.renderer.removeClass(this.document.body, t.type)
    );
    this.renderer.addClass(this.document.body, type);
    this.renderer.setAttribute(this.document.body, 'cds-theme', type);
  }

  switchTheme(): void {
    const theme = this.isLightThemeEnabled() ? 'dark' : 'light';
    this.themeType.next(theme);
    this.loadTheme();
  }

  isLightThemeEnabled(): boolean {
    return this.themeType.value === lightTheme.type;
  }
}
