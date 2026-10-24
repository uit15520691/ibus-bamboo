package main

import "testing"

func TestIsSensitiveInputContext(t *testing.T) {
	tests := []struct { name string; purpose, hints uint32; want bool }{
		{name: "normal", want: false},
		{name: "password", purpose: ibusInputPurposePassword, want: true},
		{name: "pin", purpose: ibusInputPurposePIN, want: true},
		{name: "private hint", hints: ibusInputHintPrivate, want: true},
		{name: "unrelated hint", hints: 1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSensitiveInputContext(tt.purpose, tt.hints); got != tt.want {
				t.Fatalf("isSensitiveInputContext(%d, %#x) = %v, want %v", tt.purpose, tt.hints, got, tt.want)
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
