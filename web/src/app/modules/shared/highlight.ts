/*
 * Copyright (c) 2020 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 */

import { HIGHLIGHT_OPTIONS } from 'ngx-highlightjs';

const languages = () => {
  return {
    json: () => import('highlight.js/lib/languages/json'),
    yaml: () => import('highlight.js/lib/languages/yaml'),
  };
};

export const highlightProvider = () => ({
  provide: HIGHLIGHT_OPTIONS,
  useValue: {
    // ngx-highlightjs 4 bundled the core library, so registering languages was
    // enough. From 6 the core has to be loaded explicitly — without this the
    // directive still adds its `hljs` class and renders the text, but nothing
    // is ever tokenised, so YAML shows up unhighlighted and the only clue is
    // `[HLJS] The core library was not imported!` on the console.
    coreLibraryLoader: () => import('highlight.js/lib/core'),
    languages: languages(),
  },
});
