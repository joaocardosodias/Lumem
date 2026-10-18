package usecase

import (
	"errors"
	"testing"
)

func TestPermanentIngestionError(t *testing.T) {
	rootErr := errors.New("arquivo corrompido")
	err := permanentIngestionFailure("falha critica: %w", rootErr)

	if err == nil {
		t.Fatal("esperava erro nao nil")
	}
	if err.Error() != "falha critica: arquivo corrompido" {
		t.Fatalf("mensagem inesperada: %s", err.Error())
	}
	if !errors.Is(err, rootErr) {
		t.Fatalf("esperava que errors.Is reconhecesse rootErr, obteve %v", err)
	}
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil || unwrapped.Error() != "falha critica: arquivo corrompido" {
		t.Fatalf("unwrap inesperado: %v", unwrapped)
	}
}
