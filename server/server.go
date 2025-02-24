package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var errTimeoutFailure = fmt.Errorf("gateway timeout")
var errTimeoutDatabase = fmt.Errorf("database timeout")
var dsn string = "root:root@tcp(localhost:3306)/quoteusdbrl?charset=utf8mb4&parseTime=True&loc=Local"
var db *gorm.DB

type UsdQuota struct {
	USDBRL USDBRLQuote `json:"USDBRL"`
}

type Response struct {
	Bid string `json:"bid"`
}

type USDBRLQuote struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
	gorm.Model
}

func main() {
	ConnectDB()
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePage)
	mux.Handle("/cotacao", http.HandlerFunc(GetQuote))
	http.ListenAndServe(":8080", mux)
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Aplicação para consulta de cotação do dólar, para receber a cotação acesse /cotacao"))
}

func GetQuote(w http.ResponseWriter, r *http.Request) {
	ctx, _ := context.WithTimeoutCause(context.Background(), 200*time.Millisecond, errTimeoutFailure)
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	log.Println("Request iniciada")
	defer log.Println("Request finalizada")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		switch ctx.Err() {
		case context.DeadlineExceeded:
			w.WriteHeader(http.StatusGatewayTimeout)
			http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			http.Error(w, "Erro desconhecido", http.StatusInternalServerError)
		}
	}
	log.Println("Chamada ao integrador feita com sucesso, processando resposta")
	defer res.Body.Close()
	var UsdBrl UsdQuota
	err = json.NewDecoder(res.Body).Decode(&UsdBrl)
	if err != nil {
		log.Println("Falha ao ler o corpo da resposta")
		http.Error(w, "Falha ao codificar a resposta", http.StatusBadRequest)
		return
	}
	resp := Response{Bid: UsdBrl.USDBRL.Bid}
	body, err := json.Marshal(resp)
	if err != nil {
		log.Println("Falha ao codificar a resposta")
		http.Error(w, "Falha ao codificar a resposta", http.StatusBadRequest)
		return
	}
	ctxDB, cancel := context.WithTimeoutCause(context.Background(), 10*time.Millisecond, errTimeoutDatabase)
	defer cancel()
	errCreate := SaveData(ctxDB, UsdBrl.USDBRL)
	if errCreate != nil {
		log.Println("Falha ao salvar a cotação no banco de dados: ", errCreate)
		w.WriteHeader(http.StatusFailedDependency)
		w.Write(body)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func ConnectDB() {
	data, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	db = data
	db.AutoMigrate(&USDBRLQuote{})
}

func SaveData(ctx context.Context, u USDBRLQuote) error {
	log.Println("Iniciando inserção no banco de dados")
	result := db.WithContext(ctx).Create(&u)
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return errTimeoutDatabase
	}
	return result.Error
}

func GetDatabase() *gorm.DB {
	return db
}
