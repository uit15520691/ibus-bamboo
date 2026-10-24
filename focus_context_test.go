package main

import (
	"testing"

	"ibus-bamboo/config"
	"github.com/BambooEngine/bamboo-core"
)

func TestShouldBypassInputMethod(t *testing.T) {
	tests := []struct { name string; purpose, hints uint32; want bool }{
		{name: "normal", want: false},
		{name: "digits", purpose: ibusInputPurposeDigits, want: true},
		{name: "number", purpose: ibusInputPurposeNumber, want: true},
		{name: "phone", purpose: ibusInputPurposePhone, want: true},
		{name: "url", purpose: ibusInputPurposeURL, want: true},
		{name: "email", purpose: ibusInputPurposeEmail, want: true},
		{name: "password", purpose: ibusInputPurposePassword, want: true},
		{name: "pin", purpose: ibusInputPurposePIN, want: true},
		{name: "private hint", hints: ibusInputHintPrivate, want: true},
		{name: "unrelated hint", hints: 1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldBypassInputMethod(tt.purpose, tt.hints); got != tt.want {
				t.Fatalf("shouldBypassInputMethod(%d, %#x) = %v, want %v", tt.purpose, tt.hints, got, tt.want)
			}
		})
	}
}

func TestNormalizeApplicationIdentity(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{input: "  \"Firefox\"  ", want: "Firefox"},
		{input: "org.gnome.TextEditor", want: "org.gnome.TextEditor"},
		{input: "", want: ""},
	} {
		if got := normalizeApplicationIdentity(tt.input); got != tt.want {
			t.Errorf("normalizeApplicationIdentity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func newFocusTestEngine() (*fakeEngine, *IBusBambooEngine) {
	fe := NewFakeEngine()
	cfg := config.DefaultCfg()
	inputMethod := bamboo.ParseInputMethod(cfg.InputMethodDefinitions, cfg.InputMethod)
	e := NewIbusBambooEngine("test", &cfg, fe, bamboo.NewEngine(inputMethod, cfg.Flags))
	return fe, e
}

func TestFocusIdentityChangeDiscardsComposition(t *testing.T) {
	fe, e := newFocusTestEngine()
	e.wmClasses = "old-app"
	if _, err := e.ProcessKeyEvent(uint32('t'), uint32('t'), 0); err != nil {
		t.Fatal(err)
	}
	if fe.preeditText == "" {
		t.Fatal("expected test key to create a preedit")
	}

	e.checkWmClass("new-app")
	if fe.commitText != "" {
		t.Fatalf("focus change committed old composition into new field: %q", fe.commitText)
	}
	if fe.preeditText != "" {
		t.Fatalf("preedit remained after focus change: %q", fe.preeditText)
	}
}

func TestStructuredFieldBypassesInputMethod(t *testing.T) {
	fe, e := newFocusTestEngine()
	if err := e.SetContentType(ibusInputPurposePassword, 0); err != nil {
		t.Fatal(err)
	}
	handled, err := e.ProcessKeyEvent(uint32('t'), uint32('t'), 0)
	if err != nil {
		t.Fatal(err)
	}
	if handled {
		t.Fatal("password-field key should pass through without Bamboo processing")
	}
	if fe.preeditText != "" || fe.commitText != "" {
		t.Fatalf("Bamboo modified sensitive field: preedit=%q commit=%q", fe.preeditText, fe.commitText)
	}

	if err := e.SetContentType(0, 0); err != nil {
		t.Fatal(err)
	}
	handled, err = e.ProcessKeyEvent(uint32('t'), uint32('t'), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !handled || fe.preeditText == "" {
		t.Fatalf("Bamboo did not resume after leaving sensitive field: handled=%v preedit=%q", handled, fe.preeditText)
	}
}
