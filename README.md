# IPCA SIDRA — Série histórica completa em Go

Consulta a tabela 1737, variável 2266 (número-índice do IPCA), para o Brasil e todos os meses disponíveis.

## Executar

Requer Go 1.22 ou superior. Sem dependências externas.

```bash
go run ./cmd/ipca
```

Ou compilar:

```bash
go build -o ipca ./cmd/ipca
./ipca
```

Ou instalar:

```bash
go install github.com/santannaf/ipca-sidra-go/cmd/ipca@latest
ipca
```

Arquivos gerados em `output/ipca_historico.csv` e `output/ipca_historico.json`.

## Campos

- `modelo`: `IPCA`.
- `mes_referencia`: `YYYYMM`, por exemplo `202609`.
- `ponto`: número-índice do IBGE (base dez/1993 = 100), com 10 casas decimais; **não é uma posição sequencial**.
- `fator`: **ponto do mês / ponto do mês anterior**, com 10 casas decimais.
- `ipca_realizado`: **`fator` − 1**, com 10 casas decimais. É a variação mensal em forma decimal, não em percentual.
- `ipca_realizado_anbima`: a mesma variação em percentual (**`ipca_realizado` × 100**), com 2 casas decimais, vírgula decimal e `%`, por exemplo `0,82%`.

No primeiro mês da série não há ponto anterior: `fator`, `ipca_realizado` e `ipca_realizado_anbima` são `null` em JSON e vazios em CSV. No JSON, os valores numéricos são strings, para preservar as casas decimais.

Exemplo ilustrativo: se agosto tem ponto `7000.0000000000` e setembro `7035.0000000000`, setembro terá `fator = 1.0050000000`, `ipca_realizado = 0.0050000000` e `ipca_realizado_anbima = 0,50%`.

## Precisão

Os cálculos usam `math/big.Rat` sobre o texto original recebido do IBGE. O arredondamento (10 casas, ou 2 no percentual) ocorre apenas na saída, e cada campo é arredondado a partir do valor exato, evitando erro por uso de `float64` e por arredondamento intermediário.

Nos meses mais antigos o número-índice é muito pequeno (`0.0000000076` em `197912`), e as 10 casas de `ponto` guardam poucos dígitos significativos. Nesses meses, dividir os `ponto` exportados não reproduz o `fator`, que é calculado com todas as casas recebidas do IBGE.

## Opções

```bash
go run ./cmd/ipca -output ./output -timeout 120s
```

A opção `-endpoint` permite apontar para um servidor de testes. O programa verifica meses duplicados e lacunas na série.

## Testes

```bash
go test ./...
```
