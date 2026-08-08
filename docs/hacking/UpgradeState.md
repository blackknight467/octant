# Upgrade state — Go/k8s + Angular 12→22

Working notes from this upgrade. It is **committed and pushed** to
`angular-22-k8s-036-upgrade` on the fork, in two commits: the upgrade itself
(4,623 files) and the `strictTemplates` defect fixes. CI is green on Linux, macOS and
Windows.

The application builds and runs, verified against a live EKS cluster. The Go suite is at
its pre-upgrade baseline. The Angular suite runs deterministically at 271/274. A clean
source tree builds end-to-end into a working binary — see
[It builds from source](#it-builds-from-source). See
[What remains](#what-remains) for the prioritised list of open work.

## It builds from source

Verified the way it actually matters: `rsync` of the working tree into a scratch
directory with **`node_modules`, `.angular`, `dist`, `build`, `release` and
`extraResources` excluded** — 210 MB of pure source — then the documented path from
`HACKING.md`:

```
go run build.go ci-quick     # npm ci  ->  npm run build  ->  go build -tags embedded
```

Exit 0, no errors in the log. It produced a 207 MB binary that reports its version and
git commit, and serves the embedded UI (`GET /` returns the real `index.html`, not a
404). `npm ci` resolved from `package-lock.json` alone, which is the step most likely to
rot and the one a warm `node_modules` never exercises.

Two things this surfaced:

- **`HACKING.md` was badly stale and would have blocked a fresh build.** It asked for
  "node 10.15.0 or above"; the Angular CLI enforces
  `^22.22.3 || ^24.15.0 || >=26.0.0` and hard-refuses anything else — this is not advisory,
  it exits. Go was listed as 1.15 against a `go.mod` directive of 1.26. Both corrected.
- **npm 11 blocks install scripts by default.** `npm ci` warns that 7 packages
  (`esbuild`, `@parcel/watcher`, `lmdb`, `msgpackr-extract`, `fsevents`, `core-js`,
  `electron-winstaller`) have install scripts "not yet covered by allowScripts". The build
  succeeds anyway — those are watch-mode, packaging, or optional-native concerns, and
  esbuild resolves through optional platform packages rather than its postinstall. Worth
  knowing before someone reaches for `npm approve-scripts`.

The Go half is not macOS-only, though: `./cmd/octant` cross-compiles clean with the
embedded assets for **linux/amd64, linux/arm64 and windows/amd64**, so nothing in the
441-file change breaks a non-Darwin Go build.

What genuinely remains untested off macOS/arm64 is the *npm* half — `npm ci` resolving
native optional packages (`lmdb`, `msgpackr-extract`, `@parcel/watcher`, esbuild's
platform binaries) on Linux and Windows — plus Electron packaging for the `nsis` and
`AppImage` targets. Those need a real runner.

## Backend — complete and verified

| | from | to |
|---|---|---|
| k8s.io/* | v0.29.0 | **v0.36.2** |
| helm.sh/helm/v3 | v3.14.4 | **v3.21.3** |
| go directive | 1.25.0 | **1.26.0** |
| k8s.io/kube-aggregator | v0.21.3 | v0.36.2 |

Helm gates the k8s and Go floor — pick the helm version first, then match. Only two
source changes were needed across seven k8s minor versions:

- `autoscaling/v2beta2` removed in k8s 1.26 → HPA behavior annotation now unmarshals
  into `autoscalingv2.HorizontalPodAutoscalerBehavior` (identical JSON shape).
- `corev1.ResourceRequirements` → `corev1.VolumeResourceRequirements` for PVC specs.

**v1 Endpoints → EndpointSlice** migrated in `internal/printer/service.go` and
`internal/objectstatus/service.go` (deprecated in k8s 1.33, was emitting warnings).
Lists `discovery.k8s.io/v1 EndpointSlice` by the `kubernetes.io/service-name` label;
ready-address semantics preserved by filtering on `Conditions.Ready` (nil = ready).
Verified against a live v1.35 cluster: 12 endpoints, matching `kubectl` exactly.

Go suite: **48 packages pass**, with exactly two failures, both confirmed environmental
rather than assumed:

- `TestTSGen_ComponentConfig` shells out to `prettier`. Re-run with
  `web/node_modules/.bin` on `PATH` it passes, so the failure is the missing binary.
- `Test_GetImageManifest` pulls real image manifests and compares them against hardcoded
  fixtures. The diff is only `"architecture": "amd64"` vs `"arm64"` — it fails on Apple
  silicon and passes on CI's amd64 runners.

## Frontend — Angular 22 reached

Angular **22.1.1**, Clarity **18.2.1**, CDK **22.1.1**, TypeScript **6.0.3**,
rxjs **7.8.2**, zone.js **0.16.2**. Build clean, `npm ls` peer audit clean, no
`--force` anywhere in the final tree.

### Icons: five separate defects, all resolved

Five independent defects stacked, which is why partial fixes kept looking like failures.

**Defect 1 — empty registry.** `icon.service.ts` used the legacy
`window['ClarityIcons'].has()/.add()` API. Clarity 18 still defines that global but no
longer routes it to the registry `cds-icon` reads, so the calls were silent no-ops.
Rewritten to `@cds/core`'s `addIcons([name, svg])`. Whole collections are now loaded in
`shared/icons.ts`, because shapes are requested from three sources that cannot be
enumerated statically: octant's templates, the Go backend at runtime, and Clarity's own
components (`filter-grid`, `ellipsis-vertical`, `check-circle`, `step-forward-2`).
`@cds/core`'s `sideEffects` allowlist covers only its `register.js` entrypoints, so the
optimizer treated the loader calls as pure and stripped the icon data; reading the
registry back and publishing the count creates an observable dependency it cannot remove.

**Defect 2 — bindings targeted the attribute, not the input.** Clarity 18 ships a real
Angular component, `ClrIcon`, with selector **`"clr-icon, cds-icon"`**, rendering into
isolated shadow DOM from its `@Input`s (`shape`, `size`, `direction`, `flip`, `solid`,
`status`, `inverse`, `badge`). Octant bound `[attr.shape]`. An attribute binding does not
feed a component input, so `shape` kept its default `'unknown'`, the component rendered
the unknown glyph, and it then reflected `shape="unknown"` back onto the host —
overwriting the value Angular had just written. Fixed by dropping the `attr.` prefix on
all 30 icon bindings.

The tell was that **static** `shape="cog"` worked while **bound** `[attr.shape]` did not:
a static attribute *does* bind to a matching component input. Same markup, opposite
behaviour, purely because one went through a binding.

Also fixed: `<clr-icon shape="caret down">`. `@clr/icons` accepted a two-word
`shape direction` value; CDS splits it into `shape="caret" direction="down"` (8 sites).

**Defect 3 — the `internal:` prefix, which predates this upgrade.** Go's
`SetNavigationIcon` stores the shape as `fmt.Sprintf("internal:%s", name)`, and nothing
on the client ever stripped it, so `internal:dna` matched no shape and every per-CRD
navigation entry rendered the unknown placeholder. `navigation.service.ts` now strips it
when the payload arrives. Confirmed pre-existing rather than a regression by running the
saved pre-migration binary and diffing the websocket payload — it emits the same
`internal:dna`. (The first comparison suggested otherwise; the old binary simply had not
finished CRD discovery yet, which is worth remembering when diffing two live servers.)

Per-component `ClarityIcons.addIcons()` calls were also removed from six components —
see the TestBed note below for why they are actively harmful, and `shared/icons.ts`
already registers those shapes.

**Verified against a live cluster:** 75/75 icons on the pods view and 67/67 on custom
resources render real glyphs — zero placeholders, zero unregistered shapes. The
placeholder is detectable programmatically: render a deliberately bogus shape and
compare `shadowRoot` SVG length (207 vs 448 for a real glyph), because an unresolved
shape still produces an `<svg>` and so passes any naive "did it render" check.

**This class of bug is invisible to the build, the tests, the peer audit and the Clarity
binding sweep.** `<clr-icon [attr.shape]="x">` is valid markup that compiles clean and
renders an element; whether a shape reaches a registry is not a compile-time question.
Diagnosis needed a running app — the decisive step was `ng.getOwningComponent()` on a
dev build, which showed `tab.icon === 'application'` while the DOM read
`shape="unknown"`, proving the data was intact and something downstream was overwriting.
Comparing against the pre-migration binary (kept as `build/octant.pre-k8s036.bak`) is
what established it was a regression at all, rather than pre-existing behaviour.

### Unit suite — runs again, 271/274 passing

It now completes with **no exclusions**: 274 of 274 executed, **271 pass, 3 fail**.
Previously it did not compile at all. The
earlier "random Karma disconnect" was never load: it was a **hang**, and the reason it
looked random is that Jasmine randomises spec order by default, so the victim moved
every run. `client.jasmine.random` is now `false` — without that, none of this was
reproducible.

Fixed to get there:

| problem | fix |
|---|---|
| `ClrPopoverToggleService` no longer exported by Clarity 18 | dropped from 2 specs |
| a spec still imported `@clr/icons` | → `@cds/core/icon` |
| `OverlayScrollbarsComponent` in `declarations` (25 specs) | → `OverlayscrollbarsModule` in `imports` |
| Karma retained every spec's DOM | `clearContext`/`kjhtml` now interactive-only |
| Electron 13 could not parse the bundle | Electron 13 → 43 |

`overlayscrollbars-ngx@0.5.2` is worth knowing about: its NgModule *declares*
`OverlayScrollbarsComponent`, but the published metadata predates standalone-by-default
so Angular 22's linker infers `standalone: true`. Referencing the component directly
fails whichever array you put it in — "is marked as standalone and can't be declared"
one way, "add an @NgModule annotation" the other. Import the NgModule.

Two further classes were fixed along the way:

- **`ng-reflect-*` attributes no longer exist.** `indicator.component.spec.ts` asserted
  `getAttribute('ng-reflect-shape')`; Angular 22 does not emit those. Asserting the real
  `shape` attribute passes — which also confirms the `[attr.shape]` → `[shape]` binding
  fix reflects correctly.
- **`ClarityIcons.addIcons()` in a component constructor breaks that component under
  TestBed.** It reaches into `@cds/core`'s global state before bootstrap and throws
  `Cannot read properties of undefined (reading 'indexOf')` from `window.CDS._version`,
  taking construction down with it — `input-filter` went 0/7 → 5/7 in isolation once the
  call was removed. Six components registered a single shape this way; that is a
  leftover from the `@clr/icons` era, since `shared/icons.ts` now loads whole
  collections centrally. Each removed shape was checked against a loaded collection
  first (`angle`/`times` → core, `upload`/`help`/`clipboard` → essential,
  `cluster`/`link` → technology). `indicator.component.ts` keeps its call: its
  `check-circle` / `exclamation-circle` / `times-circle` shapes are in **no** collection.

**Remaining after that pass: 19 failures, most of them `NG0100`.** The first reading was
that this was a spec-shape problem — `beforeEach` ran the first `detectChanges()`, then
each spec mutated state and checked again. Moving that first `detectChanges()` into each
spec did help (logs went 1/7 → 6/7), but it treated a symptom. The real cause is below.

### Angular 22 made `OnPush` the implicit default, and `fixture.detectChanges()` stopped re-rendering

This single change accounts for **10 of the 13** failures that survived everything above,
including every remaining `NG0100`. `ChangeDetectionStrategy` was renumbered:

```ts
enum ChangeDetectionStrategy { OnPush = 0, Eager = 1, /** @deprecated */ Default = 1 }
```

A component that declares no strategy is now **`OnPush`**, where it used to be
`CheckAlways`. `Default` survives only as a deprecated alias of the renamed `Eager`.

**The application is unaffected, and that was verified rather than assumed.** All 83
non-spec components declare a strategy explicitly, and the set is identical to the
pre-upgrade one: the same 12 files are `OnPush` before and after (diffed by path), and
the 72 that relied on the implicit default became 71 explicit `Eager` — the missing one
is a component deleted during the dead-code pass, not a changed strategy.

**Third-party libraries are also safe, and that needed checking separately.** Libraries
ship partial-compiled (`ɵɵngDeclareComponent`) and are linked by *our* Angular version, so
a new default could in principle be applied to them at link time — 112 of Clarity's 123
component declarations specify no `changeDetection` at all. Verified at runtime instead of
assumed: `ClrBadge`, `ClrCombobox` and `ClrDatagrid` all link as `onPush: false`, while
`ClrIcon`, which declares `OnPush` explicitly, links as `true`. The linker honours the
declaration-era default, so the change reaches only components compiled from source in
this repo — which is exactly the set the migration annotated.

The test harness is where it bites. `ComponentFixture.detectChanges()` calls
`detectChangesInView` on the fixture's host view, which now runs the template **only if
that view is dirty**:

```js
let shouldRefreshView = !!(mode === 0 && flags & 16);   // Dirty
shouldRefreshView ||= !!(flags & 64 && mode === 0 && ...);  // CheckAlways
```

A fixture's view is dirty when created, so the *first* `detectChanges()` still renders.
Assigning a plain field on `fixture.componentInstance` afterwards marks nothing, so every
later `detectChanges()` is a silent no-op — the DOM keeps its first value and the child's
input setter is never called a second time. Confirmed directly: after rebinding a new
object, `child.v.config.value` still read `"*text*"` while the parent held `"# header"`.

**The `NG0100`s were the same bug wearing a different mask.** `detectChanges()` skips the
refresh, then the `checkNoChanges()` that follows it runs in *exhaustive* mode, which
forces a full traversal (`shouldRefreshView ||= isExhaustiveCheckNoChanges()`). It sees
bindings that would change and reports changed-after-checked. Nothing was writing during
change detection at all — the pass that should have applied the change had been skipped.

`src/app/testing/rebind.ts` restores the pre-22 behaviour for specs that legitimately
rebind (mutate, `markForCheck()`, `detectChanges()`). It is a harness shim, deliberately
not a component change.

### Two real application bugs surfaced once the specs could actually see the DOM

Both were invisible while the specs were failing for harness reasons, and both are
user-visible.

**The resource viewer blanked for any object with no relationships.** Go marshals the
graph with `omitempty` on the `Edges` field, so a resource with no edges arrives
with no `edges` key at all — and `applyFilters()` opened with
`if (!rawNodes || !rawEdges) return { nodes: [], edges: [] }`. A graph of isolated nodes
is not an empty graph; the viewer rendered nothing instead of the single node. Missing
edges now default to `{}`.

**No step of a multi-step form could ever be opened.** Clarity queries its panels with
`@ContentChildren(ClrStepperPanel)` and **no `descendants`**, so it only matches direct
content children. `stepper.component.html` wrapped each panel in `<div class="step-item">`,
one level too deep: the panels never registered with `StepperService`, `openFirstPanel()`
had nothing to open, and the form rendered as a column of collapsed, unusable steps. The
wrapper is gone — it carried a class with no stylesheet and no other reference. This is
also why the spec looked like it was failing on a null button: the button lives inside the
step body, which is only rendered for an expanded panel.

That spec had a second, independent defect: it declared no `declarations` and imported no
`ClarityModule`, so the component compiled against an **empty directive scope**.
`clrStepper`, `clrStepButton` and `formGroup` all silently failed to attach, leaving the
buttons as plain `type="submit"` controls whose clicks did a native form submit — which is
what karma was reporting as "Some of your tests did a full page reload!".

**Both fixes are confirmed in the running app, not just under karma.**

- *Resource viewer*: the `atlantis-tests` ConfigMap in `default` is referenced by no pod,
  so its graph is a single isolated node. It now renders that node with its status panel;
  before the fix this view was blank. The premise was also checked at the wire level — the
  real Go marshaller emits `{"config":{"nodes":{...}}}` with **no `edges` key** for a
  node-only graph, which is exactly the payload `applyFilters()` was discarding.
- *Stepper*: verified with a throwaway plugin (steppers are a plugin-only component — the
  only in-tree reference is the `unmarshal.go` switch, so nothing in Octant itself renders
  one). It registers a tab printer on ConfigMaps that returns a two-step stepper. Step 1
  now renders expanded with its form and NEXT button; clicking NEXT collapses it with a
  completion tick and expands step 2 with SUBMIT. Before the fix no step could be opened
  at all. The plugin lived outside the repo and was removed afterwards.

### Chrome changed how it reports scroll offsets for `column-reverse`

The logs panel sticks to the newest entry via a single `flex-direction: column-reverse`
child. Chrome now places the scroll origin at the visual bottom: resting on the newest
line is `scrollTop === 0`, scrolling up into history yields **negative** offsets, and
positive values clamp to 0. The spec was written against the older behaviour, which
reported the same positions as positive numbers. Measured to confirm rather than inferred
(`scrollTop = -50` → `-50`; `scrollTop = 50` → `0`).

Only the reported sign changed, so the panel behaves identically on screen. One assertion
did have to be re-expressed: "position preserved when new logs arrive" cannot mean
"`scrollTop` unchanged" against a bottom-anchored origin, because appending content below
the viewport *must* move it — holding it fixed would slide the user forward onto newer
lines. The invariant is the offset from the top of the history, which the browser does
preserve exactly.

### `cds-icon` was claimed by two renderers — a real, user-visible bug

**I initially reported this as test-only. That was wrong**, and the correction matters:
the application was affected the whole time. I had checked the console for `attachShadow`
*errors* and found none — but Lit's `createRenderRoot()` is
`this.shadowRoot ?? this.attachShadow(...)`, so the second renderer silently *reuses* the
root instead of throwing. The failure mode is duplicate rendering, not an exception. Every
status indicator in every table drew **two** green check marks:

```
<cds-icon shape="check-circle">  ->  shadowRoot children: svg, STYLE, svg
```

Both of these claim the tag, and the registration is not avoidable:

- `@cds/core/icon/register.js` registers `cds-icon` as a **custom element**. Removing our
  two direct imports of it is not enough — `@cds/core/button/register.js` pulls it in
  transitively, and Octant uses `<cds-button>`.
- Clarity 18's **`ClrIcon`** Angular component, selector `clr-icon, cds-icon`, with
  `ViewEncapsulation.ExperimentalIsolatedShadowDom`.

**Fix: Octant's own templates use `<clr-icon>`.** That tag is *not* a registered custom
element, so `ClrIcon` is its only renderer. 13 tags across 8 templates. Clarity's own
internal `<cds-icon>` usages (datagrid filter, sort, overflow menus) still double-render,
but their two SVGs overlap exactly and paint correctly.

An earlier attempt at this same swap failed and was reverted, for an instructive reason:
`<cds-icon-button>` **starts with** `<cds-icon`, so a prefix-matching sed renamed its
opening tag but not `</cds-icon-button>`, breaking JIT compilation for every spec. The
working version matches `<cds-icon(?=[\s>])` and verifies open/close balance per file
afterwards.

Fixing this also took the suite from 19 failures to 13, because the same collision was
what broke `container.component.spec.ts` and cascaded into `input-filter`.

### The LogsComponent specs: all 11 pass, and none of it was Clarity's fault

These were written off twice — first as "environment-bound", then as "a Clarity control
writing back during change detection". Both readings were wrong. Every one of them was
the `detectChanges()` no-op above, plus two timing facts:

- The scroll assertions did need real layout, but the container is sized by
  overlayscrollbars, which initialises **asynchronously** — it measures 0x0 immediately
  after `whenStable()` and is fully laid out shortly after. Polling for real geometry
  (`testing/wait-for.ts`) instead of asserting against a half-laid-out element fixed it.
  There was never anything wrong with the layout engine.
- `.highlight-selected` is applied by `scrollToHighlight()`, which this upgrade moved into
  a microtask so it stops writing template-bound state during the pass that rendered the
  highlights. Synchronous assertions therefore sample too early and have to await it. The
  same ordering explains the search-wrap spec: it clicked *next* in the same tick as the
  filter change, so the deferred "select the first match" landed afterwards and reset the
  selection out from under the click.

Also verified directly against a live pod: the panel renders, streams, and its controls
work. The regex guard was exercised by typing `(`, `[`, `(foo`, `*` and `(?`
into the filter — no uncaught errors and rendering continues, where before the fix a
lone `(` threw `SyntaxError` out of the ngModel write.

### A trap in the "move detectChanges out of beforeEach" fix

That fix (see NG0100 above) was applied by script, and the script only added a
`fixture.detectChanges()` back to specs that had **none**. Specs that called
`detectChanges` *later* in the body silently lost their initial render and began
querying the DOM before it existed — surfacing as `Cannot read properties of null` or
an empty placeholder rather than anything that named the cause. Three specs were
affected (two in `input-filter`, one in `stepper`).

If this pattern is applied to more spec files, the rule is: a spec needs its own
`detectChanges()` **before its first DOM query**, not merely somewhere in the body.

Also fixed here: `input-filter.component.spec.ts` held its `LabelFilterService` stub in
a module-level `BehaviorSubject`, so each spec inherited the previous one's filters. The
stub is now rebuilt per spec.

### The cytoscape specs stopped flaking, then told the truth

`cytoscape.component.spec.ts` and `resource-viewer.component.spec.ts` waited a fixed
`setTimeout(..., 100)` for the graph to render. That is a race — `should select first
node` passed in one run and failed in the next with no code change between them. They now
poll via `src/app/testing/wait-for.ts` until the end state holds, with a cap.

That briefly made things look worse (one *more* red spec), which was the point: the old
spec passed about half the time by luck. What the polling then exposed was worth the
detour.

- `should select first node` was waiting for the wrong thing. Selection is applied from
  cytoscape's one-shot `render` event, which fires *after* the instance already reports
  its nodes — so waiting for nodes to exist sampled the graph before anything was
  selected. Waiting for the actual condition under test fixed it.
- `should show node labels` was not a rendering problem at all. Giving the host a real
  800x600 box changed nothing; the graph was empty because the component had filtered
  every node away. That is the `omitempty` edges bug above — a genuine defect the fixed
  delay had been hiding behind a timeout.

### Custom icon SVGs were clipped

The Octant logo rendered as a clipped wedge. `@clr/icons` used to normalise injected
SVGs; Clarity 18's `ClrIcon` injects the source verbatim into a span, so the logo's own
`width="106px" height="126px"` won inside a 46x46 host with `contain: layout` — you saw
its top-left corner. `scalableSvg()` in `icon.service.ts` strips intrinsic width/height
(keeping the `viewBox`) and is applied to both custom-icon paths: `IconService.load()` and
`navigation.service.ts`'s `registerCustomSvg`, so plugin-supplied icons scale too.

### The hang: a floating promise, found by bisection

`logs.component*.spec.ts` used to stall the whole run. Bisecting spec-by-spec (only
possible once `random: false` made it reproducible) landed on one spec, *should keep
scroll position even if new logs are coming in and user is not at bottom*, which did:

```ts
it('...', () => {                      // synchronous spec
  fixture.whenStable().then(() => {    // never awaited
    expect(...);
  });
});
```

The assertions ran after the spec had finished and its fixture was destroyed — which is
also why they surfaced as "Expected 0 to be greater than 0" thrown from `afterAll`
rather than as a normal failure. Awaiting them fixed the hang outright. Seven other
`whenStable().then(...)` sites existed; five were already inside `waitForAsync` (which
does await them) and two more were floating and have been converted.

Three product/test bugs came out of chasing it, worth keeping on their own merits:

- `@for (log of ...; track identifyLog($index, log))` keyed rows on
  `` `${timestamp}-${message}` ``. Log lines repeat verbatim constantly, and duplicate
  keys in `@for` are not tolerated the way `*ngFor`'s `trackBy` tolerated them. Now
  tracks `$index`, correct for an append-only list.
- `LogsComponent` opened a real pod-log stream — and therefore a real WebSocket to
  karma's own server — in unit tests. Now mocked.
- `LogsComponent` wrote template-bound state (`totalSelections`) from
  `ngAfterContentChecked`. `filterText` is now a setter that recomputes eagerly, and the
  DOM-dependent scroll reset is deferred past the change-detection pass.

`teardown: { destroyAfterEach: false }` has been removed — it was leaking every
fixture's DOM into the next spec.

`input-filter` is the one to look at next, and it has the same signature the logs hang
had: it passes 5/7 **in isolation** but 0/7 in the full run, so some earlier spec is
leaving `@cds/core`'s global state broken. Bisecting with `xit` found the logs culprit
in three runs; the same technique should work here.

## Deleted: SliderViewComponent, dead since 2020

Found while trying to verify the resize drag, and removed once confirmed unreachable by
every route:

- No template references it — `app-slider-view` appears nowhere.
- It is not in `DynamicComponentMapping`, which is the sole source for the one
  `createComponent` call site (`view-container.component.ts`). The `entryComponents` entry
  at HEAD was a pre-Ivy leftover, not evidence of use.
- The feature is orphaned on **both** sides. `content.component.ts` still assigns
  `extView` from `content.extensionComponent`, but its template never renders it — the
  `<app-slider-view>` line that did was deleted in `9d5a9f929` (2020-03-09, "Move Terminal
  component to tab"). On the Go side `component.NewExtension()` exists and is called by
  nothing outside vendor and tests, so nothing can even produce the data it would render.

The strongest evidence came from the build: the production bundle hash is **identical**
before and after deletion (`471aaa89df509826`). The optimizer had already proven it
unreachable; this only removes it from the source tree.

Removed: 316 lines across 5 files (component `.ts`/`.html`/`.scss`/`.spec.ts` plus
`slide-in-out.animation.ts`, which nothing else used), 3 declarations in
`shared.module.ts`, and 3 file entries in `tsconfig.compodoc.json`. Suite goes 276 → 274
executed with the same 3 failures.

**`SliderService` was deliberately kept** — it outlived its component and is still used by
`tabs.component.ts` to coordinate the active tab index for extension views.

This was worth doing because the dead component was not free: this upgrade spent 43
changed lines on it across 4 files — the `mwlResizeHandle` restructure, a new SCSS block,
the spec's `waitForAsync` migration, `ChangeDetectionStrategy.Eager` and
`standalone: false` — all maintaining something unreachable for six years.

## Carried debt, measured

The earlier figures in this document for strict-mode cost were wrong. Measured against the
current tree:

| setting | errors |
|---|---|
| baseline, as shipped | 0 |
| `noImplicitAny` only | 119 |
| `strictNullChecks` only | 128 |
| `strict` minus `strictPropertyInitialization` | 243 |
| **full `strict`** | **494** |
| **`strictTemplates`** | **61**, across 25 files |

`strictPropertyInitialization` alone accounts for 251 of the 494 — over half — and is the
flag Angular codebases most often opt out of, since component fields are set from inputs
and lifecycle hooks. So "strict minus that one flag" is 243, not 494.

**`strictTemplates` is the better value of the two**, and not only as type hygiene — some
of its 61 findings are real defects:

- `size="md"` on `cds-modal` in three places (quick-switcher, helper ×2). Valid sizes are
  `default | sm | lg | xl`, so those modals silently render at the default size.
- `(click)="onEnter($event)"` in quick-switcher, where `onEnter` takes no arguments.
- `trackByFn` referenced in `modal.component.html` but not defined on the component.
- Several trackBy call sites passing two arguments to one-argument functions — fallout
  from the control-flow migration; harmless at runtime, wrong as written.

Others are type-model gaps rather than bugs, and it is worth separating them: the
`ports.component.html` errors for `targetPort`/`targetPortName` look alarming, but the Go
side *does* send both fields — the TypeScript `Port` interface simply omits them. Nothing
is broken at runtime.

**The defects are now fixed; the modelling work is not. 61 → 37.**

Behavioural:

- The logs **Since** dropdown rendered each option's text from `container`, which is not
  in scope in that loop and is not a member of `LogsComponent` — it was copy-pasted from
  the Container dropdown above it. Options now show the duration label. Pre-existing: the
  `*ngFor` version at HEAD had the identical expression.
- Three `cds-modal` elements set `size="md"`. The stylesheet only carries
  `[size=sm|lg|xl]` rules, so `md` matched nothing and rendered at the default width —
  now stated as `default`, which is what they were already doing.
- quick-switcher passed `$event` to `onEnter()`, which takes no arguments.

Correctness without behaviour change: five `@for` blocks called a one-argument trackBy
with two arguments (each of those functions only returns the index, so they now track
`$index` directly), `modal.component.html` referenced a `trackByFn` the component never
defined, `NamespaceComponent.routerLinkPath` was private but called from its template,
`ModalComponent.size` is typed to cds-modal's union rather than `string`, and the `Port`
model gained the two fields Go was already sending.

The 37 that remain are all one class — type modelling. `AbstractViewComponent`'s generic
does not narrow `view` to the concrete type (so `view.config` is unknown on the base
`View`), `$event.target` is an `EventTarget`, several unions are never narrowed, and
Clarity's string-enum inputs reject plain `string`. Worth doing, but it is a modelling
project rather than a bug hunt.

### Old Angular-coupled dependencies

Four packages are compiled against Angular 12–13 and run under 22. The linker honours
their declaration-era defaults, which is why they work at all:

| package | version | compiled by | verified live |
|---|---|---|---|
| `angular-resizable-element` | 5.0.0 | 12.2.3 | yes — resize drag |
| `@materia-ui/ngx-monaco-editor` | 6.0.0 | 13.0.2 | yes — YAML tab |
| `ngx-highlightjs` | 6.1.3 | 13.2.6 | yes — **after a fix**, below |
| `overlayscrollbars-ngx` | 0.5.2 | 13.4.0 | yes — logs panel |

Monaco is the one most often flagged as risky debt, but its surface is a single template
(`editor.component.html`) and it renders correctly under Angular 22 — line numbers,
syntax colouring, minimap. Replacing it is optional, not urgent.

The other third-party rendering libraries are now accounted for too:

| library | version | status |
|---|---|---|
| `xterm` + `xterm-addon-fit` | 4.19.0 / 0.4.0 | works — pod Terminal tab attaches a live shell |
| `jsoneditor` | 9.10.5 | works — Metadata tab renders the managed-fields tree, expands on click |
| `cytoscape` (+ dagre) | 3.34.0 | works — resource viewer |
| `ansi_up` | 5.2.1 | works — logs panel |
| `d3-graphviz`, `dagre-d3`, `d3-zoom` | 2.6.1 / 0.6.4 / 2.0.0 | **unreachable** — `NewGraphviz` is constructed nowhere outside `pkg/view/component` |

The graphviz trio is the same category as the stepper was: a component type the Go side
can describe but never builds. Unlike the stepper it has no plugin-facing constructor in
use either, so those three dependencies are carried weight — worth a look if anyone is
trimming, but not a correctness risk.

### Regression found and fixed: YAML syntax highlighting

`ngx-highlightjs` was bumped **4.1.2 → 6.1.3** by this upgrade while `highlight.ts` was
left untouched. Version 4 bundled the core library, so registering languages was enough;
version 6 requires it to be loaded explicitly. Without `coreLibraryLoader` the directive
still adds its `hljs` class and renders the text, so the page looks fine — but nothing is
ever tokenised.

Caught by inspecting the DOM rather than the screenshot: `<code class="hljs">` held 5,530
characters and **zero** child spans. The only other clue was
`[HLJS] The core library was not imported!` on the console. Fixed by adding
`coreLibraryLoader: () => import('highlight.js/lib/core')`; the same element now renders
368 token spans with real `hljs-*` classes.

This affected every YAML rendered through the `yaml` view type — the Helm module's
rendered manifest and computed values, and the error component.

## What remains

Nothing here blocks the application from running; these are the open items in the order
worth doing them.

### 1. Merge the branch

Committed and pushed to `angular-22-k8s-036-upgrade`, not to `master`. Merging it is the
open decision. Opening a PR against the fork's own master would also run `lint`,
`preflight-checks` (the Go suites) and `verify-generated`, which the `electron` workflow
does not cover.

### 2. CI has now run — green on all three platforms

`origin` is `blackknight467/octant`, a public fork of the archived
`vmware-archive/octant`, and until this branch it had never run a workflow. Pushing
`angular-22-k8s-036-upgrade` triggered `electron.yaml`, which is `on: [push]` unfiltered.

**All three jobs passed: ubuntu-latest, macos-latest, windows-latest.**

That is worth more than it sounds, because of what that workflow actually does on each OS:

- `setup-node` 24.19.0 and `setup-go` 1.26.x
- `go run build.go build-electron`, which is **`npm ci` from the lockfile**, then a
  vendored Go build, then the Angular production build
- `electron-builder` packaging, including the `nsis` target on Windows and `AppImage`
  on Linux

So the two gaps this document previously called un-closable locally are closed: the build
is no longer macOS-only, and `npm ci` resolves cleanly on Linux and Windows — native
optional packages (`lmdb`, `msgpackr-extract`, `@parcel/watcher`, esbuild's platform
binaries) and all. The packaging path ran on every platform with the `.angular` exclusion
in place.

What it still does **not** cover:

- **The `upload-artifact@v4` changes.** Per-platform names and `merge-multiple` on
  download sit behind `if: startsWith(github.ref, 'refs/tags/v')` in
  `preflight-checks.yaml` and behind schedule/`workflow_dispatch` in `nightly.yaml`. Only
  a `v*` tag reaches them, and cutting a release to exercise a workflow is not worth it.
- **The test suites.** `preflight-checks` (Go tests) and `lint`/`verify-generated` only
  fire on a push to `master`/`release-*` or a PR targeting them. Opening a PR from this
  branch against the fork's own master would run them.

**Do not trigger `nightly` to test it.** It runs goreleaser with
`git tag "$(git describe --abbrev=0)+dev"` and publishes to Google Cloud Storage via
`upload-cloud-storage` — tag creation plus an external publish, which would also likely
fail on absent secrets.

### 3. Nothing here is unexercised any more

Both items that were in this slot are now done.

`electron-builder` has been run — see the packaging section; it was hiding an 11 GB app
bundle.

The resize drag is verified, though not on `slider-view` — **that component was dead and
has been deleted** (see below). The same `mwlResizeHandle` migration is used by the
resource viewer's `.gutter`, which *is* reachable, and that was dragged both directions
against a live cluster: the graph pane resizes, the status panel takes up the slack, and
cytoscape re-lays out the graph. No console errors.

The migration was also **required**, not cosmetic: `angular-resizable-element`'s changelog
records that `resizeEdges` was removed from the `mwlResizable` directive in v4, so the old
markup would have silently stopped resizing.

One caveat on method: a synthetic `left_click_drag` does nothing here — the library tracks
incremental `mousemove`s, so the drag has to be dispatched as a real
mousedown / many-mousemoves / mouseup sequence.

### 4. The 3 remaining test failures

All three are the same thing: `NotSupportedError: Failed to execute 'attachShadow' ...
already hosts a shadow tree`, in `HelperComponent`, `AppComponent` and the websocket spec.

The mechanism is now fully traced. `@cds/core/button/register.js` opens with
`import "@cds/core/icon/register.js"`, so pulling in `<cds-button>` **transitively defines
`cds-icon` as a custom element** — importing `@cds/core/icon` alone does not, which is why
this is easy to miss. Clarity's `ClrIcon` component claims the same tag
(`selector: "clr-icon, cds-icon"`) with isolated shadow DOM, and Clarity's own internal
templates emit `<cds-icon>`. Two owners, one shadow root. Whoever attaches second either
silently reuses it (Lit's `createRenderRoot()` is `this.shadowRoot ?? this.attachShadow`)
or throws (Angular's `ShadowDomRenderer` calls `attachShadow` directly) — the app hits the
first path, these specs hit the second.

It is not fixable from this side: our templates already avoid `<cds-icon>`, Clarity's do
not, and the registration cannot be dropped without giving up `<cds-button>`. The
`attachShadow` shim recovers 5–10 specs but stalls the suite around spec 270 — tried twice,
reverted twice. A completing suite at 271/274 is worth more than a hanging one.

Two earlier claims in this document were wrong and are corrected above: the logs specs were
never "environment-bound", and the `NG0100`s were never "a Clarity control writing back
during change detection". Both were the `detectChanges()` no-op. The lesson is that
`NG0100` in Angular 22 is at least as likely to mean *a render was skipped* as *something
wrote during render*.

### 5. Optional follow-ups, none of them upgrade work

Measured figures and the reasoning behind them are in
[Carried debt, measured](#carried-debt-measured); this is the short list.

| item | size | note |
|---|---|---|
| `strictTemplates` — remaining 37 | one modelling project | All one class. `AbstractViewComponent`'s generic does not narrow `view` to the concrete type, so `view.config` is unknown on the base `View`. Fixing that alone should clear roughly a third. |
| `strict: true` | 243 errors | With `strictPropertyInitialization` off, which is the usual Angular posture. 494 with it on. |
| `d3-graphviz`, `dagre-d3`, `d3-zoom` | 3 dependencies | Unreachable — `NewGraphviz` is constructed nowhere in-tree. Removable if anyone is trimming. |
| Electron package size | ~323 MB `app.asar` | Mostly `node_modules` already compiled into `dist/`. The main process needs only `electron-store`, `get-port`, `open`, `ws`; moving the frontend packages to `devDependencies` would cut most of it. Pre-existing, and it changes install semantics. |
| 3 failing specs | not fixable here | Clarity's `ClrIcon` and the `@cds` custom element both claim `<cds-icon>` and one shadow root — see section 4. |

Explicitly **not** on this list any more: Monaco, which was the most-flagged item and is
verified working under Angular 22 with a one-template surface; and the old
Angular-12/13-compiled packages, all four of which now render correctly against a live
cluster.

## What the ladder actually cost

| rung | source fixes | notes |
|---|---|---|
| 12→13 | 8 | one-time setup debt: lock regen, Node version, five years of caret drift |
| 13→14 | 0 | `FormControl` generics migrated 8 files automatically |
| 14→15 | 1 | deleted a dead 2020 `ModuleWithProviders` patch |
| 15→16 | 2 | Clarity 13→16 clean; ngcc removal survived |
| 16→17 | 1 | `--force` split Angular across two majors |
| 17→18 | 9 | Clarity 17 forced rxjs 7 + CDS theming; 4 test-only breaks |
| 19 | 0 | standalone-by-default: 100 files automatic |
| 20 | — | **skipped**: no Clarity version supports Angular 20 |
| 21 | 5 | `moduleResolution: bundler`, control-flow rewrite, trackBy arity |
| 22 | 4 | TS 6 strict default, migration conflict, `ComponentFactoryResolver` |

### Deliberately deferred (compatibility shims, not modernizations)

- `strictTemplates: false` — Angular 22 defaults it true; opting out avoids a wall of
  new errors on a codebase with no `strict` mode. The defects it reports have since been
  fixed and the remainder measured: see [Carried debt, measured](#carried-debt-measured).
- `strict: false` in `tsconfig.base.json` — **TypeScript 6.0 flipped `strict` to
  default true**. Measured cost is 494 errors, or 243 with
  `strictPropertyInitialization` off.
- Optional migrations skipped: `provideAppInitializer`, `router-current-navigation`.

### Lessons that generalise

**Peer-range width predicts upgrade cost better than maintenance activity.** Clarity
needed 4 versions across 10 Angular majors because it ships multi-major ranges
(`15||16||17||18||19`). `ng-select` needed 8 because it pins `^N.0.0` exactly. Same
job, wildly different cost.

**Published metadata misdescribes artifacts in both directions.** `ngx-monaco-editor-v2@12.0.0`
declares Angular `^12` and ships a v14 build. `angular-resizable-element@4.0.0` declares
`>=10` and ships View Engine. `@clr/angular@17.13.0` declares `15–19` but is compiled
with Angular 15.2.10, so it runs fine on 20. Check `ɵɵngDeclare` `minVersion`/`version`
in the tarball, not the peer range.

**`ng update --force` is a trap.** It cost three escalating failures, ending with
`core` at 17 while seven siblings sat at 18/19 — surfacing as a misleading error inside
`platform-browser.mjs` about missing `@angular/core` internals.

**`ng update` only pins packages you name.** Unnamed `@angular/*` resolve to latest.
Read `npm view @angular/core@N peerDependencies` first — that names `zone.js` and the
TypeScript range, both of which I missed by pattern-matching on the `@angular/` prefix.

**When a library turns an element into a component, `[attr.x]` silently stops working.**
While `clr-icon` was a global web component, binding the attribute was the only option.
Once Clarity 18 made it an Angular component, the attribute binding became a no-op that
the component then overwrote. Nothing errors — the element still renders, just wrong.
After any major upgrade, grep for `[attr.*]` on elements the library now owns.

**A control-flow migration can drop loop variables.** `*ngFor="let tab of modules; let i
= index"` became `@for (tab of modules; track identifyTab($index))` — losing `let i =
$index` while leaving a `(click)="setModule(i)"` that referenced it. Undeclared template
identifiers resolve to `undefined` rather than erroring, so module tabs silently stopped
switching. Six of seven loops kept their index var; only this one lost it, so the check
is per-loop: diff `let x = index` before against `let x = $index` after.

**Removed component *inputs* compile clean.** Clarity 17 dropped
`clrStackViewSetsize`/`clrStackViewPosinset`; 18 dead bindings survived a green build
and only appeared as `NG0303` at runtime. Sweep for them:

```
extract [clrXxx] from templates → grep against the Clarity bundle
```

## What a code review of this diff caught

Worth recording, because most of these were introduced *by* the upgrade work and none
were caught by the build or either test suite:

| defect | fix |
|---|---|
| `upload-artifact@v4` unguarded in nightly.yaml's 3-OS matrix — would have killed macOS/Windows nightlies every night | gated to `ubuntu-latest`, matching preflight-checks.yaml |
| EndpointSlice read failed the **whole** Services table for users whose RBAC predates EndpointSlice | match `*errors.AccessError`, degrade to a warning status |
| `filterText` setter compiled raw user input as a regex, so typing `(` in the logs filter threw | `safeRegex()` helper; an incomplete pattern means "no filter" |
| dual-stack Services listed every endpoint pod twice (one slice per address family) | restrict to the Service's primary IP family |
| endpoints table order came from an unsorted informer indexer | `table.Sort("IP")` |
| `prettier -c` failed on 66 files, red CI lint on every push | `npm run prettier` |
| the new ready-condition filter had no test at all | two fixtures + two cases, verified by mutation |
| ready-condition logic duplicated across two packages | extracted to `internal/util/kubernetes.EndpointReady` |

Two findings were not fixed at the time: `strictTemplates: false`, and the
`ClusterClient()` nil stub that leaves the pod-metrics columns untested (needs a
metrics-server fake, and belongs with that feature rather than this upgrade). The first has
since been revisited — the real defects behind those errors are fixed, and what is left is
measured in [Carried debt, measured](#carried-debt-measured).

The lesson that generalises: **a green build and a green suite say nothing about CI
workflow semantics, RBAC-degraded paths, or input that is invalid only momentarily.** All
three classes appeared here.

## Electron and CI

**Electron 13 → 43.3.0** (with electron-builder 22 → 26, karma-electron 6 → 7). This
was not optional: Electron 13 ships Chromium 91, which cannot parse the ES2022 *static
initialization blocks* (`static { this.ɵfac = ... }`) that Angular emits from v16 on. It
failed with a bare `Uncaught SyntaxError: Unexpected token '{'` and ran zero specs — and
the same bundle powers the desktop app, so the desktop build was equally broken.

Two API removals surfaced:

- **`File.path` was removed in Electron 32.** `select-file.component.ts` sent the
  absolute path to the backend so it could read the file directly. Replaced with
  `webUtils.getPathForFile()` via a new `ElectronService.pathForFile()`. This one is
  worth noting because the compiler caught it only because Electron's own typings
  stopped augmenting `File` — the runtime failure would have been a silent `undefined`.
- **`enableRemoteModule`** was removed in Electron 14; the option is now dead.

`electron-store` stays at v5: it is used only from the main process, so it never needed
the removed `remote` module, and v9+ is ESM-only which the CommonJS `main.ts` cannot
consume.

### Packaging swept in Angular's build cache — an 11 GB app bundle

`electron-builder` had never actually been run. It succeeds, and that is the problem: the
first run produced an **11 GB `Octant.app`** and a **1.1 GB DMG**, against a `dist/` of
94 MB.

`electron-builder.json` selects payload with `"files": ["**/*", ...exclusions]`. That list
was written for an Angular 12 tree. Angular 13 introduced `.angular/`, a **persistent
on-disk build cache**, which nothing excluded — it had grown to 10 GB here and went
straight into `app.asar`. Confirmed by listing the archive: 6,182 `.angular` entries
alongside the 1,583 that are actually the app.

**The obvious exclusion does not work, and the reason is worth knowing.** `"!.angular/"`
changes nothing. `app-builder-lib`'s `fileMatcher.js` auto-expands a bare directory
pattern to `dir/**/*` — but only when the pattern has no dot in it:

```js
// do not add if contains dot (possibly file if has extension)
if (!pattern.includes(".") && !hasMagic(parsedPattern)) {
  result.push(new Minimatch(`${pattern}/**/*`, minimatchOptions));
}
```

A leading-dot directory looks like it might be a file, so it is never expanded, and the
pattern matches only the directory entry while everything inside it still ships. (Trailing
slashes are stripped by `path.posix.normalize` first, so they make no difference either.)
This is exactly why `"!src/"` works and `"!.angular/"` does not. The pattern has to carry
its own glob: **`"!.angular/**/*"`**.

Result, measured: **11 GB → 724 MB** for the app bundle (`.angular/`, `coverage/` and
`documentation.json` excluded). `.angular/` was also missing from `web/.gitignore`, so the
cache was showing up as untracked in `git status`.

Verified beyond "the build exits 0": the packaged app launches, Electron spawns the Go
server from `Contents/Resources/extraResources/octant`, the server binds a port and
answers HTTP, and a renderer process loads `app.asar`. Building it properly needs the Go
step too — `go run build.go build-electron` compiles the server into `web/extraResources/`
*before* packaging; running `electron-builder` alone produces an app with no backend.

**This would first have appeared on a release tag.** The packaging step is guarded by
`if: startsWith(github.ref, 'refs/tags/v')`, so no PR or nightly build touches it — the
first sign would have been a multi-gigabyte DMG attached to a GitHub release.

Still oversized, but pre-existing rather than upgrade-induced: the remaining 323 MB
`app.asar` is mostly `node_modules` (`@angular`, `core-js`, `@cds`, `ace-builds`, `rxjs`,
`monaco-editor`, `lodash`), all of which are already compiled into `dist/octant`. The
Electron main process requires only four real packages — `electron-store`, `get-port`,
`open`, `ws` — so moving the frontend packages to `devDependencies` would cut most of it.
Left alone here because it predates this work and changes install semantics.

CI moved to **Node 24.19.0** (from 16) and `checkout`/`setup-node` to v4. The artifact
actions needed more than a version bump — `upload-artifact@v2` merged same-named uploads
and v4 rejects them, and both uploads sit in a 3-platform matrix:

- `bundle` is platform-independent → uploaded from `ubuntu-latest` only.
- `electron-artifacts` is per-platform → now `electron-artifacts-${{ matrix.platform }}`,
  recombined on download with `pattern:` + `merge-multiple: true`.

## Environment

Frontend assets are embedded via `//go:embed dist/octant` behind the `embedded` build
tag, so a template change normally needs a full Go rebuild to observe. For UI debugging,
run the Angular dev server against a live octant instead — it gives ~2s rebuilds and,
because it is a development build, Angular's `ng.getComponent()` / `ng.getOwningComponent()`
debug API for reading component state straight out of the running page:

```
octant --disable-origin-check --listener-addr=127.0.0.1:7825
npx ng serve --proxy-config proxy.conf.json --port 4270
# proxy.conf.json: {"/api": {"target": "http://127.0.0.1:7825", "ws": true}}
```

`--disable-origin-check` is required or octant rejects the proxied websocket upgrade and
the socket closes 1006 with no console error.

`ng update` requires Node ≥24.15. Installed **v24.19.0** via nvm alongside 24.14.1.
CI still pins Node 16 across all 5 workflows and must move before any Angular build
works there. The `--openssl-legacy-provider` workaround has been removed from
`build.go` — it was only needed for Angular 12's webpack.
