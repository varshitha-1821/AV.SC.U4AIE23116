package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger
const accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJNYXBDbGFpbXMiOnsiYXVkIjoiaHR0cDovLzIwLjI0NC41Ni4xNDQvZXZhbHVhdGlvbi1zZXJ2aWNlIiwiZW1haWwiOiJhdi5zYy51NGFpZTIzMTE2QGF2LnN0dWRlbnRzLmFtcml0YS5lZHUiLCJleHAiOjE3NzgwNTg3ODMsImlhdCI6MTc3ODA1Nzg4MywiaXNzIjoiQWZmb3JkIE1lZGljYWwgVGVjaG5vbG9naWVzIFByaXZhdGUgTGltaXRlZCIsImp0aSI6ImMzMmUwOWZlLTFmYmMtNDI3My1iMWM2LTZlZWJlY2NiM2YxMSIsImxvY2FsZSI6ImVuLUlOIiwibmFtZSI6ImphanVsYSB5b2dhIHZhcnNoaXRoYSIsInN1YiI6ImQ2ZjE4NTkyLTExZDktNDQ3OC04NDlhLTkxZTY4NmY1MmMxMiJ9LCJlbWFpbCI6ImF2LnNjLnU0YWllMjMxMTZAYXYuc3R1ZGVudHMuYW1yaXRhLmVkdSIsIm5hbWUiOiJqYWp1bGEgeW9nYSB2YXJzaGl0aGEiLCJyb2xsTm8iOiJhdi5zYy51NGFpZTIzMTE2IiwiYWNjZXNzQ29kZSI6IlBUQk1tUSIsImNsaWVudElEIjoiZDZmMTg1OTItMTFkOS00NDc4LTg0OWEtOTFlNjg2ZjUyYzEyIiwiY2xpZW50U2VjcmV0IjoiaHFVbUVmeVJ6UnpHRFhRSyJ9.c_WFP6V1XBVMPt_AofQ_JS2gxcwpnMY82mX9vMAk1fg"

func Log(stack, level, pkg, message string) {
	payload := map[string]string{
		"stack":   stack,
		"level":   level,
		"package": pkg,
		"message": message,
	}
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "http://20.207.122.201/evaluation-service/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Log error:", err)
		return
	}
	defer resp.Body.Close()
}

// Model
type Schedule struct {
	ID          int    `json:"id"`
	Vehicle     string `json:"vehicle"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Status      string `json:"status"`
}

// In-memory database
var schedules []Schedule
var nextID = 1

// Handlers
func getSchedules(c *gin.Context) {
	Log("backend", "info", "handler", "Fetching all schedules")
	c.JSON(200, schedules)
}

func getSchedule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	for _, s := range schedules {
		if s.ID == id {
			Log("backend", "info", "handler", "Fetched schedule by ID")
			c.JSON(200, s)
			return
		}
	}
	Log("backend", "error", "handler", "Schedule not found")
	c.JSON(404, gin.H{"error": "Schedule not found"})
}

func createSchedule(c *gin.Context) {
	var s Schedule
	if err := c.ShouldBindJSON(&s); err != nil {
		Log("backend", "error", "handler", "Invalid request body")
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	s.ID = nextID
	nextID++
	s.Date = time.Now().Format("2006-01-02")
	s.Status = "pending"
	schedules = append(schedules, s)
	Log("backend", "info", "handler", "Created new schedule")
	c.JSON(201, s)
}

func updateSchedule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var updated Schedule
	if err := c.ShouldBindJSON(&updated); err != nil {
		Log("backend", "error", "handler", "Invalid request body")
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	for i, s := range schedules {
		if s.ID == id {
			updated.ID = id
			schedules[i] = updated
			Log("backend", "info", "handler", "Updated schedule")
			c.JSON(200, updated)
			return
		}
	}
	Log("backend", "error", "handler", "Schedule not found for update")
	c.JSON(404, gin.H{"error": "Schedule not found"})
}

func deleteSchedule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	for i, s := range schedules {
		if s.ID == id {
			schedules = append(schedules[:i], schedules[i+1:]...)
			Log("backend", "info", "handler", "Deleted schedule")
			c.JSON(200, gin.H{"message": "Schedule deleted"})
			return
		}
	}
	Log("backend", "error", "handler", "Schedule not found for delete")
	c.JSON(404, gin.H{"error": "Schedule not found"})
}

func main() {
	r := gin.Default()

	r.GET("/schedules", getSchedules)
	r.GET("/schedules/:id", getSchedule)
	r.POST("/schedules", createSchedule)
	r.PUT("/schedules/:id", updateSchedule)
	r.DELETE("/schedules/:id", deleteSchedule)

	Log("backend", "info", "main", "Vehicle Maintenance Scheduler started")
	fmt.Println("Server running on port 8080")
	r.Run(":8080")
}