package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	logAPIURL   = "http://20.207.122.201/evaluation-service/logs"
	accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJNYXBDbGFpbXMiOnsiYXVkIjoiaHR0cDovLzIwLjI0NC41Ni4xNDQvZXZhbHVhdGlvbi1zZXJ2aWNlIiwiZW1haWwiOiJhdi5zYy51NGFpZTIzMTE2QGF2LnN0dWRlbnRzLmFtcml0YS5lZHUiLCJleHAiOjE3NzgwNTg3ODMsImlhdCI6MTc3ODA1Nzg4MywiaXNzIjoiQWZmb3JkIE1lZGljYWwgVGVjaG5vbG9naWVzIFByaXZhdGUgTGltaXRlZCIsImp0aSI6ImMzMmUwOWZlLTFmYmMtNDI3My1iMWM2LTZlZWJlY2NiM2YxMSIsImxvY2FsZSI6ImVuLUlOIiwibmFtZSI6ImphanVsYSB5b2dhIHZhcnNoaXRoYSIsInN1YiI6ImQ2ZjE4NTkyLTExZDktNDQ3OC04NDlhLTkxZTY4NmY1MmMxMiJ9LCJlbWFpbCI6ImF2LnNjLnU0YWllMjMxMTZAYXYuc3R1ZGVudHMuYW1yaXRhLmVkdSIsIm5hbWUiOiJqYWp1bGEgeW9nYSB2YXJzaGl0aGEiLCJyb2xsTm8iOiJhdi5zYy51NGFpZTIzMTE2IiwiYWNjZXNzQ29kZSI6IlBUQk1tUSIsImNsaWVudElEIjoiZDZmMTg1OTItMTFkOS00NDc4LTg0OWEtOTFlNjg2ZjUyYzEyIiwiY2xpZW50U2VjcmV0IjoiaHFVbUVmeVJ6UnpHRFhRSyJ9.c_WFP6V1XBVMPt_AofQ_JS2gxcwpnMY82mX9vMAk1fg"
)

type LogPayload struct {
	Stack   string `json:"stack"`
	Level   string `json:"level"`
	Package string `json:"package"`
	Message string `json:"message"`
}

func Log(stack, level, pkg, message string) {
	payload := LogPayload{
		Stack:   stack,
		Level:   level,
		Package: pkg,
		Message: message,
	}

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", logAPIURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Log sent! Status:", resp.Status)
}

func main() {
	Log("backend", "info", "handler", "Testing logging middleware")
	Log("backend", "error", "handler", "received string, expected bool")
	Log("backend", "fatal", "db", "Critical database connection failure.")
}