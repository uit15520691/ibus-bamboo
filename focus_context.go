package main

import "strings"

// IBus ContentPurpose values from ibusinputcontext.h.
const (
	ibusInputPurposeDigits   uint32 = 2
	ibusInputPurposeNumber   uint32 = 3
	ibusInputPurposePhone    uint32 = 4
	ibusInputPurposeURL      uint32 = 5
	ibusInputPurposeEmail    uint32 = 6
	ibusInputPurposePassword uint32 = 8
	ibusInputPurposePIN      uint32 = 9
	ibusInputHintPrivate     uint32 = 1 << 11
)

// shouldBypassInputMethod honors IBus field metadata for credentials and
// structured Latin/numeric fields. Applications that do not set content
// metadata still need their normal per-application exclusions.
func shouldBypassInputMethod(purpose, hints uint32) bool {
	switch purpose {
	case ibusInputPurposeDigits, ibusInputPurposeNumber, ibusInputPurposePhone,
		ibusInputPurposeURL, ibusInputPurposeEmail,
		ibusInputPurposePassword, ibusInputPurposePIN:
		return true
	default:
		return hints&ibusInputHintPrivate != 0
	}
}

func normalizeApplicationIdentity(identity string) string {
	return strings.TrimSpace(strings.ReplaceAll(identity, "\"", ""))
}
