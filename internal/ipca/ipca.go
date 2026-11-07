package ipca

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// Precisão dos valores numéricos exportados.
	DecimalPrecision = 10

	// Precisão do percentual apresentado no padrão ANBIMA.
	AnbimaPrecision = 2

	// Série histórica completa do IPCA.
	// Tabela 1737, variável 2266: número-índice.
	Endpoint = "https://apisidra.ibge.gov.br/values/t/1737/n1/all/v/2266/p/all/d/v2266%2013/l/v,p,t?formato=json"
)

// Record representa um registro mensal da série histórica do IPCA.
//
// Modelo: indicador econômico.
// MesReferencia: período no formato YYYYMM.
// Ponto: número-índice oficial do IBGE.
// Fator: ponto atual / ponto anterior.
// IPCARealizado: fator - 1.
// IPCARealizadoAnbima: variação percentual com duas casas decimais.
//
// No primeiro mês da série, os campos derivados são nil,
// pois não existe ponto anterior disponível.
type Record struct {
	Modelo              string  `json:"modelo"`
	MesReferencia       string  `json:"mes_referencia"`
	Ponto               string  `json:"ponto"`
	Fator               *string `json:"fator"`
	IPCARealizado       *string `json:"ipca_realizado"`
	IPCARealizadoAnbima *string `json:"ipca_realizado_anbima"`
}

// Fetch consulta a série histórica completa do IPCA no SIDRA.
func Fetch(
	ctx context.Context,
	client *http.Client,
	endpoint string,
) ([]Record, error) {

	if client == nil {
		client = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"criar requisição SIDRA: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ipca-sidra-go/4.0")

	res, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf(
			"consultar SIDRA: %w",
			err,
		)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"SIDRA retornou HTTP %d",
			res.StatusCode,
		)
	}

	var rows []map[string]string

	err = json.NewDecoder(
		io.LimitReader(res.Body, 32<<20),
	).Decode(&rows)

	if err != nil {
		return nil, fmt.Errorf(
			"decodificar JSON SIDRA: %w",
			err,
		)
	}

	if len(rows) < 2 {
		return nil, errors.New(
			"SIDRA não retornou registros de dados",
		)
	}

	// Identifica a coluna correspondente ao mês.
	periodKey := ""

	for key, label := range rows[0] {

		l := strings.ToLower(label)

		if strings.HasPrefix(key, "D") &&
			strings.HasSuffix(key, "C") &&
			(strings.Contains(l, "mês") ||
				strings.Contains(l, "mes") ||
				strings.Contains(l, "período") ||
				strings.Contains(l, "periodo")) {

			periodKey = key
			break
		}
	}

	// Fallback: identifica a coluna pelo formato YYYYMM.
	if periodKey == "" {

		for key, value := range rows[1] {

			if strings.HasPrefix(key, "D") &&
				strings.HasSuffix(key, "C") &&
				isMonth(value) {

				periodKey = key
				break
			}
		}
	}

	if periodKey == "" {
		return nil, fmt.Errorf(
			"coluna do mês não identificada: %v",
			rows[0],
		)
	}

	type item struct {
		month string
		point *big.Rat
	}

	items := make([]item, 0, len(rows)-1)

	seen := make(map[string]bool)

	// Processa os registros recebidos do IBGE.
	for i, row := range rows[1:] {

		month := strings.TrimSpace(row[periodKey])

		if !isMonth(month) {
			return nil, fmt.Errorf(
				"linha %d: mês inválido %q",
				i+2,
				month,
			)
		}

		if seen[month] {
			return nil, fmt.Errorf(
				"mês duplicado: %s",
				month,
			)
		}

		seen[month] = true

		rawValue := strings.ReplaceAll(
			strings.TrimSpace(row["V"]),
			",",
			".",
		)

		point, ok := new(big.Rat).SetString(rawValue)

		if !ok || point.Sign() <= 0 {
			return nil, fmt.Errorf(
				"linha %d (%s): ponto inválido %q",
				i+2,
				month,
				row["V"],
			)
		}

		items = append(items, item{
			month: month,
			point: point,
		})
	}

	// Ordenação cronológica crescente.
	sort.Slice(items, func(i, j int) bool {
		return items[i].month < items[j].month
	})

	result := make([]Record, 0, len(items))

	// Calcula fator, IPCA realizado e percentual ANBIMA.
	for i, current := range items {

		record := Record{
			Modelo:        "IPCA",
			MesReferencia: current.month,
			Ponto: current.point.FloatString(
				DecimalPrecision,
			),
		}

		// O primeiro período não possui ponto anterior.
		if i == 0 {
			result = append(result, record)
			continue
		}

		previous := items[i-1]

		prevDate, _ := time.Parse(
			"200601",
			previous.month,
		)

		currDate, _ := time.Parse(
			"200601",
			current.month,
		)

		// Verifica se existem lacunas na série histórica.
		if !prevDate.AddDate(0, 1, 0).Equal(currDate) {
			return nil, fmt.Errorf(
				"lacuna na série entre %s e %s",
				previous.month,
				current.month,
			)
		}

		// Fator = ponto atual / ponto anterior.
		fator := new(big.Rat).Quo(
			current.point,
			previous.point,
		)

		// IPCA realizado = fator - 1.
		ipcaRealizado := new(big.Rat).Sub(
			fator,
			big.NewRat(1, 1),
		)

		// IPCA realizado ANBIMA = IPCA realizado * 100.
		percentualAnbima := new(big.Rat).Mul(
			ipcaRealizado,
			big.NewRat(100, 1),
		)

		// Formatação dos valores com precisão configurável.
		fatorFormatado := fator.FloatString(
			DecimalPrecision,
		)

		ipcaRealizadoFormatado := ipcaRealizado.FloatString(
			DecimalPrecision,
		)

		// Percentual com duas casas decimais.
		percentualAnbimaFormatado := percentualAnbima.FloatString(
			AnbimaPrecision,
		)

		// Formatação brasileira: 0,82%.
		percentualAnbimaFormatado = strings.ReplaceAll(
			percentualAnbimaFormatado,
			".",
			",",
		) + "%"

		record.Fator = &fatorFormatado
		record.IPCARealizado = &ipcaRealizadoFormatado
		record.IPCARealizadoAnbima = &percentualAnbimaFormatado

		result = append(result, record)
	}

	return result, nil
}

// isMonth valida uma referência no formato YYYYMM.
func isMonth(s string) bool {

	if len(s) != 6 {
		return false
	}

	t, err := time.Parse("200601", s)

	return err == nil && t.Format("200601") == s
}

// WriteCSV exporta os registros para CSV.
func WriteCSV(
	path string,
	records []Record,
) error {

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return err
	}

	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer f.Close()

	w := csv.NewWriter(f)

	// Cabeçalho do CSV.
	if err := w.Write([]string{
		"modelo",
		"mes_referencia",
		"ponto",
		"fator",
		"ipca_realizado",
		"ipca_realizado_anbima",
	}); err != nil {
		return err
	}

	for _, r := range records {

		fator := ""
		ipcaRealizado := ""
		ipcaRealizadoAnbima := ""

		if r.Fator != nil {
			fator = *r.Fator
		}

		if r.IPCARealizado != nil {
			ipcaRealizado = *r.IPCARealizado
		}

		if r.IPCARealizadoAnbima != nil {
			ipcaRealizadoAnbima = *r.IPCARealizadoAnbima
		}

		if err := w.Write([]string{
			r.Modelo,
			r.MesReferencia,
			r.Ponto,
			fator,
			ipcaRealizado,
			ipcaRealizadoAnbima,
		}); err != nil {
			return err
		}
	}

	w.Flush()

	return w.Error()
}

// WriteJSON exporta os registros para JSON.
func WriteJSON(
	path string,
	records []Record,
) error {

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return err
	}

	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)

	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(records)
}
