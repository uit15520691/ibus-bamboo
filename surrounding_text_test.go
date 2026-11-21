package main

import (
	"testing"

	ibus "github.com/BambooEngine/goibus"
	"github.com/BambooEngine/ibus-bamboo/config"
	"github.com/godbus/dbus/v5"
)

func TestEffectiveBackspaceMode(t *testing.T) {
	if got := effectiveBackspaceMode(false, config.XTestFakeKeyEventIM); got != config.XTestFakeKeyEventIM {
		t.Fatalf("X11 mode changed: got %d", got)
	}
	if got := effectiveBackspaceMode(true, config.XTestFakeKeyEventIM); got != config.BackspaceForwardingIM {
		t.Fatalf("Wayland XTest mode should map to ForwardKeyEvent, got %d", got)
	}
	if got := effectiveBackspaceMode(true, config.SurroundingTextIM); got != config.SurroundingTextIM {
		t.Fatalf("Wayland surrounding-text mode should be preserved, got %d", got)
	}
}

func TestSupportsSurroundingText(t *testing.T) {
	_, e := newFocusTestEngine()
	if !e.supportsSurroundingText() {
		t.Fatal("unknown capabilities should keep the historical request behavior")
	}
	if err := e.SetCapabilities(IBusCapPreeditText); err != nil {
		t.Fatal(err)
	}
	if e.supportsSurroundingText() {
		t.Fatal("client without surrounding-text capability should not be queried")
	}
	if err := e.SetCapabilities(IBusCapPreeditText | IBusCapSurroundingText); err != nil {
		t.Fatal(err)
	}
	if !e.supportsSurroundingText() {
		t.Fatal("client with surrounding-text capability should be queried")
	}
}

func TestSurroundingTextFromVariant(t *testing.T) {
	text := ibus.NewText("Xin chào")
	got, ok := surroundingTextFromVariant(dbus.MakeVariant(*text))
	if !ok || got != "Xin chào" {
		t.Fatalf("surroundingTextFromVariant() = %q, %v; want %q, true", got, ok, "Xin chào")
	}

	got, ok = surroundingTextFromVariant(dbus.MakeVariant([]interface{}{"IBusText", uint32(0), "fallback"}))
	if !ok || got != "fallback" {
		t.Fatalf("tuple fallback = %q, %v; want %q, true", got, ok, "fallback")
	}

	if _, ok := surroundingTextFromVariant(dbus.MakeVariant("plain string")); ok {
		t.Fatal("plain string should not be treated as an IBusText payload")
	}
}
