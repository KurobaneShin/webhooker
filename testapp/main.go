package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	router := http.NewServeMux()

	router.HandleFunc("POST /payments/webhook", handlePaymentWebhook)

	http.ListenAndServe(":3000", router)
}

type WebhookRequest struct {
	Amount  int    `json:"amount"`
	Message string `json:"message"`
}

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	var req WebhookRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Fatal(err)
	}

	fmt.Println("webhook data: ", req)
}
