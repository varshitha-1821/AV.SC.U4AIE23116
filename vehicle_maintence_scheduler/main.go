package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJNYXBDbGFpbXMiOnsiYXVkIjoiaHR0cDovLzIwLjI0NC41Ni4xNDQvZXZhbHVhdGlvbi1zZXJ2aWNlIiwiZW1haWwiOiJhdi5zYy51NGFpZTIzMTE2QGF2LnN0dWRlbnRzLmFtcml0YS5lZHUiLCJleHAiOjE3NzgwNjIwNTksImlhdCI6MTc3ODA2MTE1OSwiaXNzIjoiQWZmb3JkIE1lZGljYWwgVGVjaG5vbG9naWVzIFByaXZhdGUgTGltaXRlZCIsImp0aSI6ImU3ZWQwNWNkLTdhZmYtNDY5NS1iMDUxLWVlMzdhMWM2ZTcyOSIsImxvY2FsZSI6ImVuLUlOIiwibmFtZSI6ImphanVsYSB5b2dhIHZhcnNoaXRoYSIsInN1YiI6ImQ2ZjE4NTkyLTExZDktNDQ3OC04NDlhLTkxZTY4NmY1MmMxMiJ9LCJlbWFpbCI6ImF2LnNjLnU0YWllMjMxMTZAYXYuc3R1ZGVudHMuYW1yaXRhLmVkdSIsIm5hbWUiOiJqYWp1bGEgeW9nYSB2YXJzaGl0aGEiLCJyb2xsTm8iOiJhdi5zYy51NGFpZTIzMTE2IiwiYWNjZXNzQ29kZSI6IlBUQk1tUSIsImNsaWVudElEIjoiZDZmMTg1OTItMTFkOS00NDc4LTg0OWEtOTFlNjg2ZjUyYzEyIiwiY2xpZW50U2VjcmV0IjoiaHFVbUVmeVJ6UnpHRFhRSyJ9.-i8RI37RLXzQQ6jHu3tc0e1DvqEbRq2uIVqJ6vifImo"
	baseURL     = "http://20.207.122.201/evaluation-service"
)

// Logging Middleware
func Log(stack, level, pkg, message string) {
	payload := map[string]string{
		"stack": stack, "level": level,
		"package": pkg, "message": message,
	}
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", baseURL+"/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
}

// Structs
type Depot struct {
	ID           int `json:"ID"`
	MechanicHours int `json:"MechanicHours"`
}

type Vehicle struct {
	TaskID   string `json:"TaskID"`
	Duration int    `json:"Duration"`
	Impact   int    `json:"Impact"`
}

type ScheduleResult struct {
	DepotID       int       `json:"depot_id"`
	MechanicHours int       `json:"mechanic_hours"`
	SelectedTasks []Vehicle `json:"selected_tasks"`
	TotalImpact   int       `json:"total_impact"`
	TotalDuration int       `json:"total_duration"`
}

// Fetch depots from test server
func fetchDepots() ([]Depot, error) {
	req, _ := http.NewRequest("GET", baseURL+"/depots", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Depots []Depot `json:"depots"`
	}
	json.Unmarshal(body, &result)
	return result.Depots, nil
}

// Fetch vehicles from test server
func fetchVehicles() ([]Vehicle, error) {
	req, _ := http.NewRequest("GET", baseURL+"/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Vehicles []Vehicle `json:"vehicles"`
	}
	json.Unmarshal(body, &result)
	return result.Vehicles, nil
}

// Knapsack algorithm - finds best combination of vehicles
func knapsack(vehicles []Vehicle, maxHours int) ([]Vehicle, int) {
	n := len(vehicles)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, maxHours+1)
	}

	for i := 1; i <= n; i++ {
		for w := 0; w <= maxHours; w++ {
			dp[i][w] = dp[i-1][w]
			if vehicles[i-1].Duration <= w {
				val := dp[i-1][w-vehicles[i-1].Duration] + vehicles[i-1].Impact
				if val > dp[i][w] {
					dp[i][w] = val
				}
			}
		}
	}

	// Find selected vehicles
	selected := []Vehicle{}
	w := maxHours
	for i := n; i > 0; i-- {
		if dp[i][w] != dp[i-1][w] {
			selected = append(selected, vehicles[i-1])
			w -= vehicles[i-1].Duration
		}
	}
	return selected, dp[n][maxHours]
}

// API Handler
func getSchedule(c *gin.Context) {
	Log("backend", "info", "handler", "Fetching schedule for all depots")

	depots, err := fetchDepots()
	if err != nil {
		Log("backend", "error", "handler", "Failed to fetch depots")
		c.JSON(500, gin.H{"error": "Failed to fetch depots"})
		return
	}

	vehicles, err := fetchVehicles()
	if err != nil {
		Log("backend", "error", "handler", "Failed to fetch vehicles")
		c.JSON(500, gin.H{"error": "Failed to fetch vehicles"})
		return
	}

	results := []ScheduleResult{}
	for _, depot := range depots {
		selected, totalImpact := knapsack(vehicles, depot.MechanicHours)
		totalDuration := 0
		for _, v := range selected {
			totalDuration += v.Duration
		}
		results = append(results, ScheduleResult{
			DepotID:       depot.ID,
			MechanicHours: depot.MechanicHours,
			SelectedTasks: selected,
			TotalImpact:   totalImpact,
			TotalDuration: totalDuration,
		})
		Log("backend", "info", "handler", fmt.Sprintf("Depot %d scheduled with impact %d", depot.ID, totalImpact))
	}

	c.JSON(200, gin.H{"schedules": results})
}

func main() {
	r := gin.Default()
	r.GET("/schedule", getSchedule)
	Log("backend", "info", "main", "Vehicle Maintenance Scheduler started")
	fmt.Println("Server running on port 8080")
	r.Run(":8080")
}