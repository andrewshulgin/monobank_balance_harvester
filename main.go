package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Account struct {
	Id           string   `json:"id"`
	SendId       string   `json:"sendId"`
	Balance      int      `json:"balance"`
	CreditLimit  int      `json:"creditLimit"`
	Type         string   `json:"type"`
	CurrencyCode int      `json:"currencyCode"`
	CashbackType string   `json:"cashbackType"`
	MaskedPan    []string `json:"maskedPan"`
	Iban         string   `json:"iban"`
}

type Jar struct {
	Id           string `json:"id"`
	SendId       string `json:"sendId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Balance      int    `json:"balance"`
	CurrencyCode int    `json:"currencyCode"`
	Goal         int    `json:"goal"`
}

type ClientInfo struct {
	ClientId    string    `json:"clientId"`
	Name        string    `json:"name"`
	WebHookUrl  string    `json:"webHookUrl"`
	Permissions string    `json:"permissions"`
	Accounts    []Account `json:"accounts"`
	Jars        []Jar     `json:"jars"`
}

type WebHookSetupRequest struct {
	WebHookUrl string `json:"webHookUrl"`
}

type StatementItem struct {
	Id              string `json:"id"`
	Time            int    `json:"time"`
	Description     string `json:"description"`
	Mcc             int    `json:"mcc"`
	OriginalMcc     int    `json:"originalMcc"`
	Hold            bool   `json:"hold"`
	Amount          int    `json:"amount"`
	OperationAmount int    `json:"operationAmount"`
	CurrencyCode    int    `json:"currencyCode"`
	CommissionRate  int    `json:"commissionRate"`
	CashbackAmount  int    `json:"cashbackAmount"`
	Balance         int    `json:"balance"`
	Comment         string `json:"comment"`
	ReceiptId       string `json:"receiptId"`
	InvoiceId       string `json:"invoiceId"`
	CounterEdrpou   string `json:"counterEdrpou"`
	CounterIban     string `json:"counterIban"`
}

type WebHookRequestData struct {
	Account       string        `json:"account"`
	StatementItem StatementItem `json:"statementItem"`
}

type WebHookRequest struct {
	Type string             `json:"type"`
	Data WebHookRequestData `json:"data"`
}

type AccountBalance struct {
	Id      string
	Balance int
}

var webHookUrl = os.Getenv("WEBHOOK_URL")
var token = os.Getenv("TOKEN")
var listenAddr = os.Getenv("LISTEN_ADDR")
var outputFile = os.Getenv("OUTPUT")

var accountBalances []AccountBalance

func updateBalance() {
	var sum = 0
	for _, accountBalance := range accountBalances {
		sum += accountBalance.Balance
	}
	log.Printf("Balance: %d\n", sum)
	err := os.WriteFile(
		outputFile,
		[]byte(fmt.Sprintf("%d\n", sum/100)),
		0644,
	)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}(r.Body)
	body, err := io.ReadAll(r.Body)
	w.WriteHeader(http.StatusNoContent)

	var webHookRequest WebHookRequest
	err = json.Unmarshal(body, &webHookRequest)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	if webHookRequest.Type == "StatementItem" {
		for i, accountBalance := range accountBalances {
			if webHookRequest.Data.Account == accountBalance.Id {
				accountBalances[i].Balance = webHookRequest.Data.StatementItem.Balance
			}
		}
	}
	updateBalance()
}

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.monobank.ua/personal/client-info", nil)
	req.Header.Add("X-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	var clientInfo ClientInfo
	err = json.Unmarshal(body, &clientInfo)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}

	for _, account := range clientInfo.Accounts {
		if account.Type == "white" && account.CurrencyCode == 980 {
			accountBalances = append(accountBalances, AccountBalance{account.Id, account.Balance})
		}
	}
	for _, jar := range clientInfo.Jars {
		accountBalances = append(accountBalances, AccountBalance{jar.Id, jar.Balance})
	}

	log.Printf("Accounts: %+v\n", accountBalances)
	updateBalance()

	if clientInfo.WebHookUrl != webHookUrl {
		req, err = http.NewRequest("GET", "https://api.monobank.ua/personal/webhook", nil)
		req.Header.Add("X-Token", token)
		resp, err = client.Do(req)
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}

	http.HandleFunc("/", handler)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
}
