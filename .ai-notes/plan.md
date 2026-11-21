# ibus-bamboo improvement plan (Ubuntu 26.10 / GNOME Shell 51 / IBus 1.5.34)

Created: 2026-11-04
Target environment (verified on this machine):

- Ubuntu 26.10, Wayland session (`XDG_SESSION_TYPE=wayland`), `XDG_CURRENT_DESKTOP=ubuntu:GNOME`
- GNOME Shell 51.0
- IBus 1.5.34-1ubuntu3
- GTK 4.24.1 (setup UI is still GTK3)
- Engine currently installed: locally built `0.8.5-local`, binary md5 matches repo `ibus-engine-bamboo`
- `bamboo-focus@ibus-bamboo` extension: installed in `~/.local/share/gnome-shell/extensions/`, **ACTIVE but stale** (Oct 22 copy; repo has Oct 24 `get_gtk_application_id()` fallback)

Goal: make the engine reliable on modern GNOME/Wayland, make the build reproducible and
versioned, and remove fragile/legacy paths. Work is split into phases so each can be
tested independently.

Working rules:

1. Do not break existing X11, FreeBSD, or wlroots behavior.
2. Keep all existing user changes (`README.md`, `scripts/build`) unless a step explicitly
   replaces them.
3. Test with `make docker-test` / `make docker-build` (Docker is available) or native
   `go test -mod=vendor ./...` when GTK3 headers are installed.
4. Commit-sized changes per phase; update the checklist at the bottom as work lands.
5. Heavy items that cannot be completed safely in one pass (GTK4 rewrite, gettext,
   emoji/dictionary data refresh) are documented as follow-ups rather than half-done.

---

## Phase 1 — GNOME 51 / Wayland integration (P0)

### 1.1 Ship and install the focus extension

Problem: `scripts/install`, `build/deb`, `Makefile`, and the RPM spec never install
`gnome-extension/`. On GNOME 51, `org.gnome.Shell.Eval` is disabled, so without the
extension every user loses per-application exclusions and per-application input modes.

Changes:

- `scripts/install`:
  - install `gnome-extension/extension.js` and `metadata.json` into
    `${PREFIX}/share/gnome-shell/extensions/bamboo-focus@ibus-bamboo/`.
  - keep `DESTDIR` support for packaging.
- `Makefile`:
  - `install-extension`: copy into
    `$(HOME)/.local/share/gnome-shell/extensions/bamboo-focus@ibus-bamboo/`,
    then print the `gnome-extensions enable ...` command.
  - `pack-extension`: `gnome-extensions pack gnome-extension -o build/` for
    extensions.gnome.org / manual distribution.
  - Document that system-wide extensions still must be enabled by the user.
- `build/rpm/ibus-bamboo.spec`: add `%{_datadir}/gnome-shell/extensions/...` to `%files`.
- `build/deb/rules` already runs `make install PREFIX=/usr`, so the script change covers
  the `.deb`.

Acceptance: `make install DESTDIR=/tmp/pkg PREFIX=/usr` creates
`/tmp/pkg/usr/share/gnome-shell/extensions/bamboo-focus@ibus-bamboo/{extension.js,metadata.json}`.

### 1.2 Update extension metadata and remove the dead Shell Eval path

Changes:

- `gnome-extension/metadata.json`:
  - `version: 2`, `version-name` from the project version,
  - `shell-version: ["51", "52"]` (forward compatible; 51 already covers 51.x),
  - `session-modes: ["user", "unlock-dialog"]`,
  - `url` pointing at the repository.
- `gnome-extension/extension.js`: no functional change required for this pass; keep the
  existing `get_wm_class() || get_gtk_application_id()` fallback.
- `gnome_introspector.go`:
  - delete `gnomeGetFocusWindowClassWithEval` and `gnomeIsOverviewVisibleWithEval`;
  - return an actionable error that tells the user to install/enable the extension;
  - drop the now-unused `errors` import.
- `gnome-extension/README.md`: remove the sentence claiming Bamboo falls back to Shell
  Eval; document that the extension is the only supported GNOME path.

Acceptance: `go build` succeeds; no `org.gnome.Shell.Eval` string remains in the unit
except in documentation explaining why it is not used.

### 1.3 Capability-gated surrounding text

Problem: `RequireSurroundingText()` is emitted unconditionally (`engine.go:125,157`), while
`SetCapabilities` only stores the bitmask and never uses it. Clients that cannot provide
surrounding text get pointless requests; the engine can then run the fragile reconstruction
path.

Changes:

- `engine.go`: store capabilities atomically; add `supportsSurroundingText()` helper.
- Call `RequireSurroundingText()` from `FocusIn`/`Enable` only when
  `capabilities & IBusCapSurroundingText != 0`.
- `SetSurroundingText` remains guarded by `isSurroundingTextReady`.

Acceptance: a unit test can set/clear capabilities and assert the helper result; existing
tests still pass.

### 1.4 Wayland-safe correction mode

Problem: `XTestFakeKeyEventIM` uses XTest, which only reaches XWayland clients. On a
native Wayland GNOME session it will not correct text in GTK4/Qt6 apps.

Changes:

- `main.go`: detect Wayland from `WAYLAND_DISPLAY` **or** `XDG_SESSION_TYPE=wayland`;
  keep GNOME detection but also accept `GNOME_DESKTOP_SESSION_ID`.
- `engine_backspace.go`: in `SendBackSpace`, if `isWayland` and the configured mode is
  `XTestFakeKeyEventIM`, log a warning and use the `ForwardKeyEvent` backspace path.
- Optionally log the effective mode once at engine start.
- `wl_introspector.go`: add a comment that `zwlr_foreign_toplevel_manager_v1` is a
  wlroots-only protocol and is intentionally skipped on GNOME (extension handles it).

Acceptance: code path is unit-testable by factoring the mode selection into a helper
(`effectiveBackspaceMode(isWayland, configured)`).

---

## Phase 2 — Build, version, packaging (P1)

### 2.1 Single source of truth for the version

Problem: version `0.8.5` is duplicated in `Makefile`, `scripts/build`, `scripts/install`,
`data/bamboo.xml`, `flake.nix`, `build/deb/changelog`, `build/deb/control`, and the RPM
spec. This is why local builds could not be distinguished from apt builds.

Changes:

- Add `VERSION` (base release, `0.8.5`).
- Add executable `scripts/version`:
  - `BAMBOO_VERSION` env override wins (useful for distro builds);
  - otherwise `<VERSION>+<short-sha>[.dirty]` inside a git checkout;
  - otherwise plain `<VERSION>`.
- `scripts/build`: use `$(sh scripts/version)` for `-X main.Version=...`; keep
  `BAMBOO_VERSION` override.
- `Makefile`: `version = $(shell sh scripts/version)`; fix the malformed `GOLDFLAGS`
  line; use `version` for tar names/spec; add `PREFIX ?= /usr`.
- `scripts/install`: replace the hardcoded `version=`; render `data/bamboo.xml` with the
  build version and the actual `${PREFIX}` paths (this also fixes 2.5).
- `flake.nix`: read `VERSION` instead of hardcoding.
- `data/bamboo.xml`: keep the base version as a checked-in default; installed copies are
  rendered by `scripts/install`.
- `build/deb/control`: fix `Standards-Version: 0.8.5` → `4.7.0`; drop the explicit
  `libgtk-3-0` dependency (let `${shlibs:Depends}` handle the t64 transition on Ubuntu
  24.04+). Keep `ibus` as a dependency.

Acceptance: `sh scripts/version` prints a stable value; `./ibus-engine-bamboo --version`
matches it after a build; `docker-build` no longer produces different version strings for
the same commit.

### 2.2 Non-root Docker builds

Problem: `make docker-build` mounts the repo and runs as root, leaving `root:root`
artifacts (observed on `ibus-engine-bamboo`).

Changes:

- `Makefile docker-build`: add `--user "$(shell id -u):$(shell id -g)"`,
  `GOCACHE=/tmp/go-cache`, `GOMODCACHE=/tmp/go-mod-cache`.
- Keep `Dockerfile.test` as-is; the mounted source is writable by the host user.

Acceptance: a fresh `make docker-build` produces a binary owned by the invoking user.

### 2.3 Test against the real target (Ubuntu 26.10 / IBus 1.5.34)

Changes:

- Add `Dockerfile.ubuntu` based on `ubuntu:26.10` with Go, build-essential, pkg-config,
  libx11-dev, libxtst-dev, libgtk-3-dev, libibus-1.0-dev.
- Add `make docker-test-ubuntu` (build + run tests) and `make docker-build-ubuntu`.
- If `ubuntu:26.10` is not available in the registry at build time, the target must fail
  with a clear message rather than silently using an older base.

Acceptance: `docker build -f Dockerfile.ubuntu -t ibus-bamboo-ubuntu .` succeeds and the
container runs `make test && make build`.

### 2.4 CI hygiene

Changes to `.github/workflows/test.yml`:

- use `go test -mod=vendor -race ./...`;
- add `go vet -mod=vendor ./...`;
- add a `gofmt -l` check (fail if output is non-empty);
- cache modules only from `go.sum`/vendor (no `go mod` mutation).

Acceptance: workflow passes on the repo, and locally `go vet`/`gofmt` are clean (the two
currently unformatted files are fixed as part of Phase 3).

### 2.5 Module modernization

Changes:

- `go.mod`: `module github.com/BambooEngine/ibus-bamboo`, `go 1.24`.
- Update internal imports: `ibus-bamboo/config` → `github.com/BambooEngine/ibus-bamboo/config`,
  same for `ui`.
- Keep `vendor/` (no network dependency required to build).
- Replace `io/ioutil` with `os` in `engine_utils.go`, `config/config.go`, `ui/ui.go`.
- `initConfigFiles`: `os.MkdirAll(dir, 0755)` instead of `os.Mkdir(dir, 0777)`.

Acceptance: `go build -mod=vendor` and `go test -mod=vendor ./...` pass in Docker.

---

## Phase 3 — Runtime robustness and typing latency (P2)

### 3.1 Debug-only logging

Problem: `ProcessKeyEvent` prints on every key (`engine.go:106-107`), and many hot paths
use unconditional `fmt.Printf`/`log.Printf`. This costs latency and spams the journal.

Changes:

- Add `debug.go` with `debugf(...)` gated by `BAMBOO_DEBUG=1` (also `BAMBOO_DEBUG=0`
  disables). Keep real errors on `log.Printf`.
- Convert hot-path per-key logs in `engine.go`, `engine_backspace.go`, `engine_utils.go`,
  and the `WM_CLASS` print in `FocusIn` to `debugf`.
- Keep focus-identity warnings (once per focus change) as normal logs.

Acceptance: normal typing produces no per-key output; `BAMBOO_DEBUG=1` restores it.

### 3.2 Data-race fixes

Problems:

- `dictionary` and `emojiTrie` are package globals written from `FocusIn`/`PropertyActivate`
  and read from the key-processing goroutine.
- `wlAppId` is written by the Wayland introspection goroutine and read on focus.
- `bypassInputMethod`, `contentPurpose`, `contentHints` are written from D-Bus methods and
  read from `ProcessKeyEvent`.
- `wmClasses` is written in `FocusIn` and read from the key goroutine.

Changes:

- Protect `dictionary` with `sync.RWMutex` + accessors; update `engine_preedit.go` reader.
- Protect `emojiTrie` with `sync.RWMutex` + accessors; update `emoji.go` readers.
- Replace `wlAppId` with `atomic.Value` (string).
- Use `atomic.Bool`/`atomic.Uint32` for content type fields; keep behavior identical.
- Protect `wmClasses` with a dedicated `sync.RWMutex`.
- Run `go test -race -mod=vendor ./...` in Docker.

Acceptance: `-race` tests pass; no direct access to the protected globals remains.

### 3.3 Remove panics from runtime paths

Changes:

- `engine.go`: emoji load failure → log and continue (emoji lookup disabled).
- `engine_utils.go`: config dir/macro template failure → log and continue with defaults.
- `ui/ui.go`: config/macro read/write failure → log and return; never `panic`.
- `config/config.go`: already non-panicking; keep.

Acceptance: removing `data/emojione.json` or making the config dir read-only does not
crash the engine.

### 3.4 Robust `SetSurroundingText` decoding

Problem: `engine.go:181` extracts the text with
`reflect.ValueOf(reflect.ValueOf(text.Value()).Index(2).Interface()).String()`, which
depends on `goibus.Text` field order and silently falls into `recover()` on any change.

Changes:

- Add a small struct mirroring `goibus.Text` and decode via `dbus.Variant.Store`.
- Fall back to the old slice-shape extraction only if `Store` fails; return without
  touching the preeditor if neither works.
- Bounds-check `cursorPos` against the decoded rune length.
- Add a unit test with `dbus.MakeVariant(ibus.Text{Name:"IBusText", Text:"..."})` plus a
  malformed variant.

Acceptance: test covers the real serialized shape; no `reflect` import remains in
`engine.go`.

---

## Phase 4 — UX / data (P3, scoped)

### 4.1 Fix `C.CString` leaks in the setup UI

`ui/ui.go:100` allocates two C strings and never frees them. Allocate, `defer C.free`,
then call `C.openGUI(...)`.

### 4.2 Deferred with rationale (not in this pass)

- **GTK4/libadwaita port of the setup UI**: the GUI is ~18 KB of GTK3 C
  (`ui/keyboard-shortcut-editor.c`). It needs a port/rewrite and visual testing on GNOME
  51; doing it blind risks breaking configuration. Track as a separate project.
- **gettext/i18n**: requires extracting every user-facing string from C and Go,
  adding `.po` files and build integration. Do after the GTK4 decision so strings are
  extracted once.
- **Emoji/dictionary refresh**: replace EmojiOne (2017) and the 2018 Vietnamese word list.
  Needs a data-format decision and licensing review. Not a code-risk item.

### 4.3 Small UX improvements in this pass

- Add `BAMBOO_DEBUG` documentation to `README.md` troubleshooting.
- Keep the existing `--version` output format stable (single line).

---

## Phase 5 — Packaging/documentation corrections (P4)

### 5.1 README regression

The uncommitted README change wraps two `sudo install` commands in backticks inside a
fenced code block, so they are copied literally. Remove the backticks.

### 5.2 Install paths

Render `data/bamboo.xml` at install time with `${PREFIX}` (handled by 2.1). This fixes
`make install PREFIX=/usr/local` and non-Debian prefixes; the Nix flake can drop its
`sed` workaround.

### 5.3 RPM spec

- Install the GNOME extension (1.1).
- Keep the existing `%files` list in sync with `scripts/install`.

---

## Verification matrix

| Check | Command | Expected |
|---|---|---|
| Format | `gofmt -l .` (excluding vendor) | empty |
| Vet | `go vet -mod=vendor ./...` | clean |
| Unit tests | `make docker-test` | pass |
| Race tests | `docker run ... go test -race -mod=vendor ./...` | pass |
| Version | `./ibus-engine-bamboo --version` | equals `sh scripts/version` |
| Install tree | `make install DESTDIR=/tmp/pkg PREFIX=/usr` | binary, XML, icons, extension present |
| XML paths | `grep /usr /tmp/pkg/usr/share/ibus/component/bamboo.xml` | all paths under `/usr` |
| No Eval | `grep -rn org.gnome.Shell.Eval --include='*.go' .` | only docs/comments |
| Normal logging | run engine, type | no per-key output |
| Debug logging | `BAMBOO_DEBUG=1` | per-key logs return |

## Risks / rollback

- Removing Shell Eval could regress ancient GNOME sessions; they are outside the target
  and the extension install path is now packaged. Rollback: restore the two helper
  functions from git history.
- Capability gating may leave surrounding text disabled if a client never calls
  `SetCapabilities`; IBus does call it before focus. If a regression appears, fall back to
  requesting surrounding text when the capability is unknown (0).
- Version strings switch from `0.8.5` to `0.8.5+<sha>` for local builds; release tarballs
  built from a clean tag should override `BAMBOO_VERSION=0.8.5` in CI if needed.
- `atomic` field types change struct layout; no serialization depends on it.

## Progress checklist

- [x] Plan written
- [x] 1.1 Extension packaged/installed (`scripts/install`, `make install-extension`, RPM `%files`)
- [x] 1.2 Metadata + remove Eval (`metadata.json`, `gnome_introspector.go`, extension README)
- [x] 1.3 Capability-gated surrounding text (`SetCapabilities`, `requireSurroundingText`)
- [x] 1.4 Wayland-safe backspace (`effectiveBackspaceMode`, session detection)
- [x] 2.1 Single version source (`VERSION`, `scripts/version`, install-time XML rendering)
- [x] 2.2 Non-root Docker build (`--user`, `HOME`/cache overrides)
- [x] 2.3 Ubuntu 26.10 Docker target (`Dockerfile.ubuntu`, `make docker-test-ubuntu`)
- [x] 2.4 CI hygiene (`-mod=vendor -race`, vet, gofmt)
- [x] 2.5 go.mod modernization (module path, go 1.24, `os` APIs, re-vendored)
- [x] 3.1 Debug-only logging (`debug.go`, hot paths converted)
- [x] 3.2 Race fixes (dictionary/emoji/wlAppId/content-type/wmClasses)
- [x] 3.3 Panic removal (engine, config files, setup UI)
- [x] 3.4 SetSurroundingText decode (typed `dbus.Variant.Store` + fallback + tests)
- [x] 4.1 C string leaks (`ui/ui.go`)
- [x] 5.1 README fix (backticks removed)
- [x] 5.2 Prefix-aware XML install (`render_component`)
- [x] 5.3 RPM files update

Deferred (documented in 4.2): GTK4/libadwaita setup UI, gettext/i18n,
EmojiOne/dictionary data refresh.

## Verification log

- `docker build -f Dockerfile.ubuntu -t ibus-bamboo-ubuntu .` → OK (Ubuntu 26.10, IBus 1.5.34)
- `go test -mod=vendor ./...` in Ubuntu 26.10 container → OK
- `go test -mod=vendor -race ./...` in bookworm test image → OK
- `go vet -mod=vendor ./...` in Ubuntu 26.10 container → OK
- `make docker-build` → binary owned by the invoking user
- `./ibus-engine-bamboo --version` → `0.8.5+39d28d4.dirty` (matches `sh scripts/version`)
- `make install DESTDIR=... PREFIX=/usr` → binary, XML, icons and GNOME extension installed
- `make install DESTDIR=... PREFIX=/usr/local` → XML `exec`/`icon` paths rendered under `/usr/local`
- `make pack-extension` → produces `build/bamboo-focus@ibus-bamboo.shell-extension.zip`
- `make install-extension` → updated the user-local extension; GNOME keeps reporting
  metadata `Version: 1` until the Shell is restarted, but the D-Bus API is active and
  returns the focused window class

Local builds now report `0.8.5+<sha>[.dirty]`. Tagged checkouts report the tag
(`v0.8.5` → `0.8.5`), and `BAMBOO_VERSION` overrides both for release/distro builds.
