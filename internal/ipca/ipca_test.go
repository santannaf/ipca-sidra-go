package ipca

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchSortPrecisionAndRatio(t *testing.T) {
	payload := `[{"V":"Valor","D1C":"Variável (Código)","D2C":"Mês (Código)"},{"V":"100.1234567891234","D1C":"2266","D2C":"202602"},{"V":"99.1","D1C":"2266","D2C":"202601"}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(payload)) }))
	defer server.Close()
	got, err := Fetch(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].MesReferencia != "202601" || got[0].Ponto != "99.10000000" || got[0].IPCARealizado != nil || got[1].Ponto != "100.12345679" || got[1].IPCARealizado == nil || *got[1].IPCARealizado != "1.01032752" {
		t.Fatalf("resultado inesperado: %+v", got)
	}
}
func TestMissingMonth(t *testing.T) {
	payload := `[{"V":"Valor","D1C":"Mês (Código)"},{"V":"100","D1C":"202601"},{"V":"101","D1C":"202603"}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(payload)) }))
	defer server.Close()
	_, err := Fetch(context.Background(), server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "lacuna") {
		t.Fatalf("esperava erro de lacuna: %v", err)
	}
}
