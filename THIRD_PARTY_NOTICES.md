# Third-party notices

Webframe's own code is MIT licensed (see `LICENSE`). The repository also ships
the third-party code and assets below. Where a row says
"UNVERIFIED — see PROVENANCE", the repository holds no license file or banner
for that item, so its license and copyright line have not been confirmed from
upstream yet; `static/js/PROVENANCE.md` records what is known.

Go module dependencies (`go.mod`) are not vendored and are not listed here.

| Component | Where in this repo | Upstream | License | Copyright |
|---|---|---|---|---|
| templUI components (v1.3.0) | `components/{aspectratio,avatar,badge,button,card,checkbox,dialog,dropdown,form,icon,input,label,popover,radio,selectbox,separator,tabs,textarea}` (files headed `// templui component ...`), `utils/templui.go`, and the paired scripts `static/js/{avatar,checkbox,dialog,dropdown,input,label,popover,selectbox,tabs,textarea}.min.js` (several locally modified, see PROVENANCE) | https://templui.io, https://github.com/templui/templui | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| `filterbar.min.js`, `slideover.min.js` | `static/js/` | unknown | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| htmx 2.0.4 | `static/js/htmx.min.js` | https://github.com/bigskysoftware/htmx | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| Alpine.js 3.14.8 | `static/js/alpine.min.js` | https://github.com/alpinejs/alpine | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| 3d-force-graph 1.73.4 | `static/js/3d-force-graph.min.js` | https://github.com/vasturiano/3d-force-graph | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| Lucide icons 0.544.0 | `components/icon/icon_data.go` (SVG path data) | https://lucide.dev, https://github.com/lucide-icons/lucide | UNVERIFIED — see PROVENANCE | UNVERIFIED — see PROVENANCE |
| Game Icons | `static/icons/game-icons.svg` (used by `components/gameicon`) | https://game-icons.net | CC BY 3.0 (https://creativecommons.org/licenses/by/3.0/), per the sprite's header | Authors: Lorc, Delapouite, Skoll, Willdabeast, Carl Olsen (https://game-icons.net/about.html), per the sprite's header |
| Tailwind CSS 4.3.3 (generated base styles) | `static/css/output.css` | https://tailwindcss.com | MIT, per the file's `/*! tailwindcss v4.3.3 \| MIT License */` banner | UNVERIFIED — see PROVENANCE |
| Inter | `static/css/fonts/inter-latin.woff2` | https://github.com/rsms/inter | SIL Open Font License 1.1 (full text: `static/css/fonts/LICENSE-Inter.txt`) | Copyright (c) 2016 The Inter Project Authors (https://github.com/rsms/inter) |
| JetBrains Mono | `static/css/fonts/jetbrains-mono-latin.woff2` | https://github.com/JetBrains/JetBrainsMono | SIL Open Font License 1.1 (full text: `static/css/fonts/LICENSE-JetBrainsMono.txt`) | Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono) |
