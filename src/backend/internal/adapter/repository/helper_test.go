package repository

import (
	"testing"
)

func TestParseTime(t *testing.T) {
	if got := parseTime(""); !got.IsZero() {
		t.Fatalf("esperava tempo zero para string vazia")
	}

	if got := parseTime("2026-09-01 15:04:05"); got.IsZero() {
		t.Fatalf("esperava parsing valido para formato datetime")
	}

	if got := parseTime("2026-09-01T15:04:05Z"); got.IsZero() {
		t.Fatalf("esperava parsing valido para RFC3339 layout")
	}

	if got := parseTime("2026-09-01"); got.IsZero() {
		t.Fatalf("esperava parsing valido para formato data simples")
	}

	if got := parseTime("formato-invalido"); !got.IsZero() {
		t.Fatalf("esperava tempo zero para formato desconhecido")
	}
}
