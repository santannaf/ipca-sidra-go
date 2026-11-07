package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"example.com/ipca-sidra-go/internal/ipca"
)

func main() {
	output := flag.String("output", "output", "diretório de saída")
	endpoint := flag.String("endpoint", ipca.Endpoint, "URL da API SIDRA (útil para testes)")
	timeout := flag.Duration("timeout", 120*time.Second, "tempo limite da consulta HTTP")
	flag.Parse()
	client := &http.Client{Timeout: *timeout}
	records, err := ipca.Fetch(context.Background(), client, *endpoint)
	if err != nil {
		log.Fatal(err)
	}
	csvPath := filepath.Join(*output, "ipca_historico.csv")
	jsonPath := filepath.Join(*output, "ipca_historico.json")
	if err := ipca.WriteCSV(csvPath, records); err != nil {
		log.Fatal(err)
	}
	if err := ipca.WriteJSON(jsonPath, records); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Exportados %d pontos (%s a %s)\nCSV: %s\nJSON: %s\n", len(records), records[0].MesReferencia, records[len(records)-1].MesReferencia, csvPath, jsonPath)
	_ = os.Stdout
}
