# GNOME Shell 51 focus detection for IBus Bamboo

On GNOME Wayland, IBus Bamboo needs the focused application's WM class (or GTK application ID when WM_CLASS is unavailable) to preserve
per-application input-mode settings and exclusions. Recent GNOME Shell releases
disable the `org.gnome.Shell.Eval` D-Bus method in normal sessions. This extension
provides a narrow D-Bus API for the focused window and does not enable arbitrary
JavaScript evaluation.

## Install for the current user

From the repository root:

```sh
install_dir="$HOME/.local/share/gnome-shell/extensions/bamboo-focus@ibus-bamboo"
mkdir -p "$install_dir"
cp gnome-extension/extension.js gnome-extension/metadata.json "$install_dir/"
gnome-extensions enable bamboo-focus@ibus-bamboo
```

If the enable command reports that the extension is not recognized, log out and
back in, then enable it from GNOME Extensions. Check status with:

```sh
gnome-extensions info bamboo-focus@ibus-bamboo
```

Restart Bamboo/IBus after enabling the extension:

```sh
ibus restart
```

The Go client tries the extension API first and falls back to the legacy Shell
Eval method for older/custom sessions that still allow it. On other desktops,
the existing Wayland/X11 introspection paths remain unchanged.

## D-Bus API

- Bus name: `org.ibus.bamboo.GnomeFocus`
- Object path: `/org/ibus/bamboo/GnomeFocus`
- Interface: `org.ibus.bamboo.GnomeFocus`
- `GetFocusedWindowClass() -> s` (returns WM_CLASS first, then GTK application ID for native Wayland windows)
- `IsOverviewVisible() -> b`

If identity lookup fails, Bamboo logs the failure and does not fall back to X11 data on Wayland; this avoids applying a stale app's exclusion or input-mode mapping.

This extension targets GNOME Shell 51. It should be tested in a real GNOME
Wayland session before treating the combination as fully supported.
