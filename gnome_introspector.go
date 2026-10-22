package main

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	gnomeFocusBusName = "org.ibus.bamboo.GnomeFocus"
	gnomeFocusPath    = dbus.ObjectPath("/org/ibus/bamboo/GnomeFocus")
	gnomeFocusIface   = "org.ibus.bamboo.GnomeFocus"
)

// gnomeGetFocusWindowClass asks the Bamboo GNOME Shell extension for the
// focused window's WM class. GNOME Shell's Eval method is disabled in normal
// sessions on current GNOME releases, so it is retained only as a backwards
// compatibility fallback for older installations.
func gnomeGetFocusWindowClass() (string, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	obj := conn.Object(gnomeFocusBusName, gnomeFocusPath)
	var class string
	err = obj.Call(gnomeFocusIface+".GetFocusedWindowClass", 0).Store(&class)
	if err == nil {
		if class == "" {
			visible, visibleErr := gnomeIsOverviewVisible(conn)
			if visibleErr == nil && visible {
				return "org.gnome.Overview", nil
			}
		}
		return class, nil
	}

	// Keep compatibility with older setups that still permit org.gnome.Shell.Eval.
	class, legacyErr := gnomeGetFocusWindowClassWithEval(conn)
	if legacyErr == nil {
		return class, nil
	}
	return "", fmt.Errorf("GNOME focus detection unavailable: extension call failed (%v); legacy Eval failed (%w). Install and enable the Bamboo GNOME Shell extension", err, legacyErr)
}

func gnomeIsOverviewVisible(conn *dbus.Conn) (bool, error) {
	obj := conn.Object(gnomeFocusBusName, gnomeFocusPath)
	var visible bool
	err := obj.Call(gnomeFocusIface+".IsOverviewVisible", 0).Store(&visible)
	return visible, err
}

func gnomeGetFocusWindowClassWithEval(conn *dbus.Conn) (string, error) {
	obj := conn.Object("org.gnome.Shell", "/org/gnome/Shell")
	var ok bool
	var result string
	jsCode := "global.get_window_actors().find(window => !Main.overview.visible && window.meta_window.has_focus()).get_meta_window().get_wm_class()"
	if err := obj.Call("org.gnome.Shell.Eval", 0, jsCode).Store(&ok, &result); err != nil {
		return "", err
	}
	if !ok {
		if visible, err := gnomeIsOverviewVisibleWithEval(conn); err == nil && visible {
			return "org.gnome.Overview", nil
		}
		if result == "" {
			result = "GNOME Shell refused JavaScript evaluation"
		}
		return "", errors.New(result)
	}
	return result, nil
}

func gnomeIsOverviewVisibleWithEval(conn *dbus.Conn) (bool, error) {
	obj := conn.Object("org.gnome.Shell", "/org/gnome/Shell")
	var ok bool
	var result string
	err := obj.Call("org.gnome.Shell.Eval", 0, "Main.overview.visible").Store(&ok, &result)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New(result)
	}
	return result == "true", nil
}
