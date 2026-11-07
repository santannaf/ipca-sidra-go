package ipca

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	want := []Record{
		{Modelo: "IPCA", MesReferencia: "202601", Ponto: "99.1000000000"},
		{
			Modelo:              "IPCA",
			MesReferencia:       "202602",
			Ponto:               "100.1234567891",
			Fator:               ptr("1.0103275155"),
			IPCARealizado:       ptr("0.0103275155"),
			IPCARealizadoAnbima: ptr("1,03%"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resultado inesperado:\nobtido:   %s\nesperado: %s", toJSON(t, got), toJSON(t, want))
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

func ptr(s string) *string { return &s }

func toJSON(t *testing.T, records []Record) string {
	t.Helper()
	b, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
