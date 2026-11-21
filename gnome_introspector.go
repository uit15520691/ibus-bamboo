package main

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	gnomeFocusBusName = "org.ibus.bamboo.GnomeFocus"
	gnomeFocusPath    = dbus.ObjectPath("/org/ibus/bamboo/GnomeFocus")
	gnomeFocusIface   = "org.ibus.bamboo.GnomeFocus"
)

// gnomeGetFocusWindowClass asks the bundled GNOME Shell extension for the
// focused window's WM class. GNOME Shell's org.gnome.Shell.Eval method is
// disabled in normal sessions on current GNOME releases, so the extension is
// the only supported way to identify the focused application on GNOME.
func gnomeGetFocusWindowClass() (string, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	obj := conn.Object(gnomeFocusBusName, gnomeFocusPath)
	var class string
	if err := obj.Call(gnomeFocusIface+".GetFocusedWindowClass", 0).Store(&class); err != nil {
		return "", fmt.Errorf(
			"GNOME focus detection unavailable: cannot call %s (%w); "+
				"install and enable the bundled bamboo-focus@ibus-bamboo GNOME Shell extension "+
				"(see gnome-extension/README.md)",
			gnomeFocusBusName, err)
	}
	if class == "" {
		visible, visibleErr := gnomeIsOverviewVisible(conn)
		if visibleErr == nil && visible {
			return "org.gnome.Overview", nil
		}
	}
	return class, nil
}

func gnomeIsOverviewVisible(conn *dbus.Conn) (bool, error) {
	obj := conn.Object(gnomeFocusBusName, gnomeFocusPath)
	var visible bool
	err := obj.Call(gnomeFocusIface+".IsOverviewVisible", 0).Store(&visible)
	return visible, err
}
