# Third-party notices

Webframe's own code is MIT licensed (see `LICENSE`). The repository also ships
the third-party code and assets below. Every license and copyright line here
was read from the upstream project's own LICENSE file at the version shipped
(checked 2026-11-02); `static/js/PROVENANCE.md` records how each script in
`static/js/` was matched to its upstream file.

Go module dependencies (`go.mod`) are not vendored and are not listed here.
First-party scripts in `static/js/` (`filterbar.min.js`, `graphwidget.min.js`,
`trafficstream.min.js` and the unminified `.js` files) are webframe's own code
under its MIT license.

## Summary

| Component | Where in this repo | Upstream | Version | License | Copyright |
|---|---|---|---|---|---|
| templUI components | `components/{aspectratio,avatar,badge,button,card,checkbox,dialog,dropdown,form,icon,input,label,popover,radio,selectbox,separator,tabs,textarea}` (files headed `// templui component ...`), `utils/templui.go`, `static/js/{avatar,checkbox,dialog,dropdown,input,label,popover,selectbox,tabs,textarea}.min.js` | https://github.com/templui/templui | v1.3.0 | MIT | Copyright (c) 2023 Axel Adrian |
| `slideover.min.js` (adapted from templUI `dialog.min.js`) | `static/js/slideover.min.js` | https://github.com/templui/templui | v1.3.0 base, locally modified | MIT | Copyright (c) 2023 Axel Adrian |
| Floating UI (`@floating-ui/core`, `@floating-ui/dom`), bundled inside templUI's popover script | `static/js/popover.min.js` | https://github.com/floating-ui/floating-ui | 1.7.0 (both packages; per the `// https://cdn.jsdelivr.net/npm/@floating-ui/...@1.7.0` headers of templUI v1.3.0 `internal/components/popover/floating_ui_{core,dom}.js`) | MIT | Copyright (c) 2021-present Floating UI contributors |
| htmx | `static/js/htmx.min.js` | https://github.com/bigskysoftware/htmx | 2.0.4 | 0BSD | none (0BSD carries no copyright line and requires no attribution; listed for completeness) |
| Alpine.js | `static/js/alpine.min.js` | https://github.com/alpinejs/alpine | 3.14.8 | MIT | Copyright © 2019-2021 Caleb Porzio and contributors |
| 3d-force-graph (plus the libraries bundled into it, next table) | `static/js/3d-force-graph.min.js` | https://github.com/vasturiano/3d-force-graph | 1.73.4 | MIT | Copyright (c) 2017 Vasco Asturiano |
| Lucide icons (SVG path data) | `components/icon/icon_data.go`, `components/icon/icon_defs.go` | https://github.com/lucide-icons/lucide | 0.544.0 | ISC; portions derived from Feather are MIT | Lucide Contributors 2025; Copyright (c) 2013-2023 Cole Bemis (Feather portions) |
| Tailwind CSS (generated base styles) | `static/css/output.css` | https://github.com/tailwindlabs/tailwindcss | 4.3.3 | MIT | Copyright (c) Tailwind Labs, Inc. |
| Game Icons | `static/icons/game-icons.svg` (used by `components/gameicon`) | https://game-icons.net | not versioned | CC BY 3.0, per the sprite's header | Authors: Lorc, Delapouite, Skoll, Willdabeast, Carl Olsen (https://game-icons.net/about.html), per the sprite's header |
| Inter | `static/css/fonts/inter-latin.woff2` | https://github.com/rsms/inter | not recorded | SIL Open Font License 1.1 | Copyright (c) 2016 The Inter Project Authors (https://github.com/rsms/inter) |
| JetBrains Mono | `static/css/fonts/jetbrains-mono-latin.woff2` | https://github.com/JetBrains/JetBrainsMono | not recorded | SIL Open Font License 1.1 | Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono) |

### Libraries bundled inside `3d-force-graph.min.js`

The 3d-force-graph 1.73.4 bundle (published to npm 2024-09-25) inlines its
whole dependency tree and carries no per-library banners. The versions below
are what its declared ranges resolved to on that publish date (`npm install
--before=2024-09-25T21:10:00Z 3d-force-graph@1.73.4`); three.js r168 matches
the `REVISION` string inside the bundle, the rest are inferred from the publish
date. The minifier may have dropped unused code from some of these (likely
`@babel/runtime`, `regenerator-runtime`, `lodash-es`); they are listed so
nothing that might be present goes unattributed.

| Library | Version | Upstream | License | Copyright |
|---|---|---|---|---|
| three (three.js r168) | 0.168.0 | https://github.com/mrdoob/three.js | MIT | Copyright © 2010-2024 three.js authors |
| three-forcegraph | 1.41.14 | https://github.com/vasturiano/three-forcegraph | MIT | Copyright (c) 2017 Vasco Asturiano |
| three-render-objects | 1.29.5 | https://github.com/vasturiano/three-render-objects | MIT | Copyright (c) 2018 Vasco Asturiano |
| kapsule | 1.14.5 | https://github.com/vasturiano/kapsule | MIT | Copyright (c) 2017 Vasco Asturiano |
| accessor-fn | 1.5.1 | https://github.com/vasturiano/accessor-fn | MIT | Copyright (c) 2017 Vasco Asturiano |
| data-joint | 1.3.1 | https://github.com/vasturiano/data-joint | MIT | Copyright (c) 2019 Vasco Asturiano |
| index-array-by | 1.4.2 | https://github.com/vasturiano/index-array-by | MIT | Copyright (c) 2018 Vasco Asturiano |
| d3-force-3d | 3.0.5 | https://github.com/vasturiano/d3-force-3d | MIT | Copyright (c) 2017 Vasco Asturiano |
| d3-binarytree | 1.0.2 | https://github.com/vasturiano/d3-binarytree | MIT | Copyright (c) 2017 Vasco Asturiano |
| d3-octree | 1.0.2 | https://github.com/vasturiano/d3-octree | MIT | Copyright (c) 2017 Vasco Asturiano |
| d3-array | 3.2.4 | https://github.com/d3/d3-array | ISC | Copyright 2010-2023 Mike Bostock |
| internmap | 2.0.3 | https://github.com/mbostock/internmap | ISC | Copyright 2021 Mike Bostock |
| d3-scale | 4.0.2 | https://github.com/d3/d3-scale | ISC | Copyright 2010-2021 Mike Bostock |
| d3-scale-chromatic | 3.1.0 | https://github.com/d3/d3-scale-chromatic | ISC; ColorBrewer schemes Apache-2.0 | Copyright 2010-2024 Mike Bostock; Copyright 2002 Cynthia Brewer, Mark Harrower, and The Pennsylvania State University |
| d3-color | 3.1.0 | https://github.com/d3/d3-color | ISC | Copyright 2010-2022 Mike Bostock |
| d3-interpolate | 3.0.1 | https://github.com/d3/d3-interpolate | ISC | Copyright 2010-2021 Mike Bostock |
| d3-format | 3.1.0 | https://github.com/d3/d3-format | ISC | Copyright 2010-2021 Mike Bostock |
| d3-time | 3.1.0 | https://github.com/d3/d3-time | ISC | Copyright 2010-2022 Mike Bostock |
| d3-time-format | 4.1.0 | https://github.com/d3/d3-time-format | ISC | Copyright 2010-2021 Mike Bostock |
| d3-dispatch | 3.0.1 | https://github.com/d3/d3-dispatch | ISC | Copyright 2010-2021 Mike Bostock |
| d3-quadtree | 3.0.1 | https://github.com/d3/d3-quadtree | ISC | Copyright 2010-2021 Mike Bostock |
| d3-timer | 3.0.1 | https://github.com/d3/d3-timer | ISC | Copyright 2010-2021 Mike Bostock |
| ngraph.forcelayout | 3.3.1 | https://github.com/anvaka/ngraph.forcelayout | BSD-3-Clause | Copyright (c) 2013 - 2022, Andrei Kashcha |
| ngraph.graph | 20.0.1 | https://github.com/anvaka/ngraph.graph | BSD-3-Clause | Copyright (c) 2013 - 2022, Andrei Kashcha |
| ngraph.events | 1.2.2 | https://github.com/anvaka/ngraph.events | BSD-3-Clause | Copyright (c) 2013-2022 Andrei Kashcha |
| ngraph.random | 1.1.0 | https://github.com/anvaka/ngraph.random | BSD-3-Clause | Copyright (c) 2013-2020, Andrei Kashcha |
| ngraph.merge | 1.0.0 | https://github.com/anvaka/ngraph.merge | MIT | Copyright (c) 2013-2019 Andrei Kashcha |
| tinycolor2 | 1.6.0 | https://github.com/bgrins/TinyColor | MIT | Copyright (c), Brian Grinstead, http://briangrinstead.com |
| @tweenjs/tween.js | 25.0.0 | https://github.com/tweenjs/tween.js | MIT | Copyright (c) 2010-2012 Tween.js authors; easing equations Copyright (c) 2001 Robert Penner |
| polished | 4.3.1 | https://github.com/styled-components/polished | MIT | Copyright (c) 2016-Present Brian Hough and Maximilian Stoiber |
| lodash-es | 4.17.21 | https://github.com/lodash/lodash | MIT | Copyright OpenJS Foundation and other contributors <https://openjsf.org/> |
| @babel/runtime | 7.25.6 | https://github.com/babel/babel | MIT | Copyright (c) 2014-present Sebastian McKenzie and other contributors |
| regenerator-runtime | 0.14.1 | https://github.com/facebook/regenerator | MIT | Copyright (c) 2014-present, Facebook, Inc. |

## License texts

### MIT License

Applies to webframe itself and to the components above marked MIT. Copyright
holders:

- webframe: see `LICENSE`
- templUI (and `slideover.min.js`, adapted from it): Copyright (c) 2023 Axel Adrian
- Floating UI (`@floating-ui/core`, `@floating-ui/dom` 1.7.0): Copyright (c) 2021-present Floating UI contributors
- Alpine.js: Copyright © 2019-2021 Caleb Porzio and contributors
- Tailwind CSS: Copyright (c) Tailwind Labs, Inc.
- Lucide (portions derived from Feather): Copyright (c) 2013-2023 Cole Bemis
- 3d-force-graph: Copyright (c) 2017 Vasco Asturiano
- three (three.js r168): Copyright © 2010-2024 three.js authors
- three-forcegraph: Copyright (c) 2017 Vasco Asturiano
- three-render-objects: Copyright (c) 2018 Vasco Asturiano
- kapsule: Copyright (c) 2017 Vasco Asturiano
- accessor-fn: Copyright (c) 2017 Vasco Asturiano
- data-joint: Copyright (c) 2019 Vasco Asturiano
- index-array-by: Copyright (c) 2018 Vasco Asturiano
- d3-force-3d: Copyright (c) 2017 Vasco Asturiano
- d3-binarytree: Copyright (c) 2017 Vasco Asturiano
- d3-octree: Copyright (c) 2017 Vasco Asturiano
- ngraph.merge: Copyright (c) 2013-2019 Andrei Kashcha
- tinycolor2: Copyright (c), Brian Grinstead, http://briangrinstead.com
- @tweenjs/tween.js: Copyright (c) 2010-2012 Tween.js authors; easing equations Copyright (c) 2001 Robert Penner
- polished: Copyright (c) 2016-Present Brian Hough and Maximilian Stoiber
- lodash-es: Copyright OpenJS Foundation and other contributors <https://openjsf.org/>
- @babel/runtime: Copyright (c) 2014-present Sebastian McKenzie and other contributors
- regenerator-runtime: Copyright (c) 2014-present, Facebook, Inc.

```
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### ISC License

Applies to Lucide and the d3 packages and internmap above. Copyright holders:

- Lucide: Copyright (c) for portions of Lucide are held by Cole Bemis 2013-2023 as part of Feather (MIT). All other copyright (c) for Lucide are held by Lucide Contributors 2025.
- d3-array: Copyright 2010-2023 Mike Bostock
- internmap: Copyright 2021 Mike Bostock
- d3-scale: Copyright 2010-2021 Mike Bostock
- d3-scale-chromatic: Copyright 2010-2024 Mike Bostock
- d3-color: Copyright 2010-2022 Mike Bostock
- d3-interpolate: Copyright 2010-2021 Mike Bostock
- d3-format: Copyright 2010-2021 Mike Bostock
- d3-time: Copyright 2010-2022 Mike Bostock
- d3-time-format: Copyright 2010-2021 Mike Bostock
- d3-dispatch: Copyright 2010-2021 Mike Bostock
- d3-quadtree: Copyright 2010-2021 Mike Bostock
- d3-timer: Copyright 2010-2021 Mike Bostock

```
Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
```

### BSD 3-Clause License

Applies to ngraph.forcelayout, ngraph.graph, ngraph.events and ngraph.random
(bundled in `3d-force-graph.min.js`). Copyright holders:

- ngraph.forcelayout: Copyright (c) 2013 - 2022, Andrei Kashcha
- ngraph.graph: Copyright (c) 2013 - 2022, Andrei Kashcha
- ngraph.events: Copyright (c) 2013-2022 Andrei Kashcha
- ngraph.random: Copyright (c) 2013-2020, Andrei Kashcha

```
<copyright line above>
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

  Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

  Redistributions in binary form must reproduce the above copyright notice, this
  list of conditions and the following disclaimer in the documentation and/or
  other materials provided with the distribution.

  Neither the name of the Andrei Kashcha nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### Zero-Clause BSD (0BSD)

Applies to htmx 2.0.4.

```
Permission to use, copy, modify, and/or distribute this software for
any purpose with or without fee is hereby granted.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES
OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE
FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY
DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN
AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT
OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
```

### Apache License 2.0 (ColorBrewer)

The ColorBrewer color schemes inside d3-scale-chromatic (bundled in
`3d-force-graph.min.js`): "Apache-Style Software License for ColorBrewer
software and ColorBrewer Color Schemes. Copyright 2002 Cynthia Brewer, Mark
Harrower, and The Pennsylvania State University. Licensed under the Apache
License, Version 2.0." Full text: https://www.apache.org/licenses/LICENSE-2.0

### SIL Open Font License 1.1

Applies to the Inter and JetBrains Mono fonts. The full license text ships
next to each font: `static/css/fonts/LICENSE-Inter.txt` and
`static/css/fonts/LICENSE-JetBrainsMono.txt`.

### Creative Commons Attribution 3.0 (CC BY 3.0)

`static/icons/game-icons.svg` contains icons from https://game-icons.net by
Lorc, Delapouite, Skoll, Willdabeast and Carl Olsen, used under CC BY 3.0
(https://creativecommons.org/licenses/by/3.0/). The icons are combined into a
single SVG sprite of `<symbol>` elements.
