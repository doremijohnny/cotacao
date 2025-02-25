package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Descobri o método "WithTimeoutCause" no pacote "context" que permite passar um erro customizado para o timeout, dessa forma, consigo identificar o motivo do timeout
var errTimeoutFailure = fmt.Errorf("gateway timeout")

type Quote struct {
	Bid string `json:"bid"`
}

func client() {
	f, err := os.Create("arquivo.txt")
	if err != nil {
		panic(err)
	}
	ctx, _ := context.WithTimeoutCause(context.Background(), 300*time.Millisecond, errTimeoutFailure)
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	// Adicionei esse opicional para passar o timeout como argumento, pois percebi que o tempo de resposta do servidor era varíavel, dessa forma, consigo simular todos os cenários sem precisar alterar o código
	// Sei que esse tipo de pratica jamais poderia ser feita em produção, mas para o propósito do exercicio, acredito que seja válido
	if len(os.Args) > 1 {
		query := req.URL.Query()
		query.Add("timeout", os.Args[1])
		req.URL.RawQuery = query.Encode()
	}
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	var quote Quote
	err = json.NewDecoder(res.Body).Decode(&quote)
	if err != nil {
		panic(err)
	}
	if _, err := fmt.Fprintf(f, "Dólar: %s\n", quote.Bid); err != nil {
		panic(err)
	}
}

func main() {
	client()
}
