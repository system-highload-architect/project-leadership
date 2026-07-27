package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func sendJSON(url string, data interface{}) {
	b, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		fmt.Println("Error sending:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Sidecar responded with status %d\n", resp.StatusCode)
	}
}

func main() {
	sidecarMetricsURL := "http://localhost:8081/metrics"
	sidecarLogsURL := "http://localhost:8082/logs" // обычно один порт, но разделим для ясности

	// Отправляем метрику
	metric := map[string]interface{}{
		"name":      "orders_processed",
		"value":     42,
		"labels":    map[string]string{"service": "order-api"},
		"timestamp": time.Now(),
	}
	sendJSON(sidecarMetricsURL, metric)

	// Отправляем лог
	logEntry := map[string]interface{}{
		"level":     "info",
		"message":   "User 123 placed an order",
		"fields":    map[string]interface{}{"user_id": 123, "order_id": "ord-456"},
		"timestamp": time.Now(),
	}
	sendJSON(sidecarLogsURL, logEntry)

	fmt.Println("Data sent to sidecar")
}
