# IPCA SIDRA — Série histórica completa em Go

Consulta a tabela 1737, variável 2266 (número-índice do IPCA), para o Brasil e todos os meses disponíveis.

## Executar

Requer Go instalado. Sem dependências externas.

```bash
go run ./cmd/ipca
```

Ou compilar:

```bash
go build -o ipca ./cmd/ipca
./ipca
```

Arquivos gerados em `output/ipca_historico.csv` e `output/ipca_historico.json`.

## Campos

- `modelo`: `IPCA`.
- `mes_referencia`: `YYYYMM`, por exemplo `202609`.
- `ponto`: número-índice original do IBGE, com 8 casas decimais; **não é uma posição sequencial**.
- `ipca_realizado`: **ponto do mês / ponto do mês anterior**, com 8 casas decimais, **sem multiplicar por 100 e sem subtrair 1**. Para o primeiro mês, `null` em JSON e campo vazio em CSV.

Exemplo ilustrativo: se agosto tem ponto `7000.00000000` e setembro `7035.00000000`, setembro terá `ipca_realizado = 1.00500000`.

**Atenção:** a taxa percentual de inflação seria `(ipca_realizado - 1) × 100`. A razão exportada aqui **não** é a taxa percentual.

## Precisão

Os cálculos usam `math/big.Rat` sobre o texto original recebido do IBGE. O arredondamento para 8 casas ocorre apenas na saída, evitando erro por uso de `float64` e por arredondamento intermediário.

## Opções

```bash
go run ./cmd/ipca -output ./output -timeout 120s
```

A opção `-endpoint` permite apontar para um servidor de testes. O programa verifica meses duplicados e lacunas na série.

## Testes

```bash
go test ./...
```
