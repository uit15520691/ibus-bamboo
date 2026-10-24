package main

import "strings"

const (
	ibusInputPurposePassword uint32 = 8
	ibusInputPurposePIN      uint32 = 9
	ibusInputHintPrivate     uint32 = 1 << 11
)

func isSensitiveInputContext(purpose, hints uint32) bool {
	return purpose == ibusInputPurposePassword ||
		purpose == ibusInputPurposePIN ||
		hints&ibusInputHintPrivate != 0
}

func normalizeApplicationIdentity(identity string) string {
	return strings.TrimSpace(strings.ReplaceAll(identity, "\"", ""))
}
