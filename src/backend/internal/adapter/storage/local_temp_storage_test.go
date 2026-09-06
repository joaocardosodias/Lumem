package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLocalTempStorageStoreOpenAndDelete(t *testing.T) {
	st := NewLocalTempStorage()
	ctx := context.Background()
	content := "col1,col2\nval1,val2\n"

	key, hash, err := st.Store(ctx, "dados.csv", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Store retornou erro: %v", err)
	}
	defer func() {
		_ = os.Remove(key)
	}()

	expectedHash := sha256.Sum256([]byte(content))
	if hash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("hash diferente do esperado: %s", hash)
	}

	rc, err := st.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open retornou erro: %v", err)
	}
	data, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil || string(data) != content {
		t.Fatalf("conteudo recuperado invalido: %s, err: %v", string(data), err)
	}

	if err := st.Delete(ctx, key); err != nil {
		t.Fatalf("Delete retornou erro: %v", err)
	}
	if _, err := os.Stat(key); !os.IsNotExist(err) {
		t.Fatalf("esperava arquivo removido, obteve erro: %v", err)
	}
}

func TestLocalTempStorageStoreDefaultExtension(t *testing.T) {
	st := NewLocalTempStorage()
	ctx := context.Background()

	key, _, err := st.Store(ctx, "dados_sem_extensao", strings.NewReader("teste"))
	if err != nil {
		t.Fatalf("Store retornou erro: %v", err)
	}
	defer st.Delete(ctx, key)

	if !strings.HasSuffix(key, ".csv") {
		t.Fatalf("esperava extensao .csv padrao: %s", key)
	}
}
