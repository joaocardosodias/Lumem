package adapter_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/g04/backend/internal/adapter/storage"
)

func TestLocalTempStorage_StoreWithExtensionAndOpen(t *testing.T) {
	s := storage.NewLocalTempStorage()
	ctx := context.Background()
	content := "sample csv content,1,2,3"
	path, hash, err := s.Store(ctx, "telemetria.csv", strings.NewReader(content))
	if err != nil {
		t.Fatalf("falha ao armazenar arquivo: %v", err)
	}
	defer s.Delete(ctx, path)

	if hash == "" {
		t.Fatalf("hash nao deveria ser vazio")
	}

	r, err := s.Open(ctx, path)
	if err != nil {
		t.Fatalf("falha ao abrir arquivo: %v", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil || string(data) != content {
		t.Fatalf("conteudo divergente ou erro: %v", err)
	}
}

func TestLocalTempStorage_StoreWithoutExtensionDefaultsToCSV(t *testing.T) {
	s := storage.NewLocalTempStorage()
	ctx := context.Background()
	path, _, err := s.Store(ctx, "dados_sem_extensao", strings.NewReader("sem-ext"))
	if err != nil {
		t.Fatalf("falha ao armazenar sem extensao: %v", err)
	}
	defer s.Delete(ctx, path)

	if !strings.HasSuffix(path, ".csv") {
		t.Fatalf("esperava extensao .csv padrao, obteve %s", path)
	}
}

func TestLocalTempStorage_DeleteFile(t *testing.T) {
	s := storage.NewLocalTempStorage()
	ctx := context.Background()
	path, _, err := s.Store(ctx, "deletar.txt", strings.NewReader("apagar"))
	if err != nil {
		t.Fatalf("falha ao armazenar: %v", err)
	}

	if err := s.Delete(ctx, path); err != nil {
		t.Fatalf("falha ao deletar: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("arquivo deveria ter sido removido")
	}
}

func TestLocalTempStorage_StoreReaderError(t *testing.T) {
	s := storage.NewLocalTempStorage()
	ctx := context.Background()
	errReader := iotest.ErrReader(errors.New("simulated read error"))
	if _, _, err := s.Store(ctx, "falha.csv", errReader); err == nil {
		t.Fatalf("esperava erro de leitura durante copia")
	}
}

func TestLocalTempStorage_StoreCreateTempError(t *testing.T) {
	t.Setenv("TMPDIR", "/nonexistent_dir_for_test")
	s := storage.NewLocalTempStorage()
	ctx := context.Background()
	if _, _, err := s.Store(ctx, "falha.csv", strings.NewReader("dados")); err == nil {
		t.Fatalf("esperava erro ao criar arquivo temporario")
	}
}
