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
type Notification struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// In-memory database
var notifications []Notification
var nextID = 1

// Handlers
func getNotifications(c *gin.Context) {
	Log("backend", "info", "handler", "Fetching all notifications")
	c.JSON(200, notifications)
}

func getNotification(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	for _, n := range notifications {
		if n.ID == id {
			Log("backend", "info", "handler", "Fetched notification by ID")
			c.JSON(200, n)
			return
		}
	}
	Log("backend", "error", "handler", "Notification not found")
	c.JSON(404, gin.H{"error": "Notification not found"})
}

func createNotification(c *gin.Context) {
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		Log("backend", "error", "handler", "Invalid request body")
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	n.ID = nextID
	nextID++
	n.Status = "pending"
	n.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	notifications = append(notifications, n)
	Log("backend", "info", "handler", "Created new notification")
	c.JSON(201, n)
}

func updateNotification(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var updated Notification
	if err := c.ShouldBindJSON(&updated); err != nil {
		Log("backend", "error", "handler", "Invalid request body")
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	for i, n := range notifications {
		if n.ID == id {
			updated.ID = id
			updated.CreatedAt = n.CreatedAt
			notifications[i] = updated
			Log("backend", "info", "handler", "Updated notification")
			c.JSON(200, updated)
			return
		}
	}
	Log("backend", "error", "handler", "Notification not found for update")
	c.JSON(404, gin.H{"error": "Notification not found"})
}

func deleteNotification(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	for i, n := range notifications {
		if n.ID == id {
			notifications = append(notifications[:i], notifications[i+1:]...)
			Log("backend", "info", "handler", "Deleted notification")
			c.JSON(200, gin.H{"message": "Notification deleted"})
			return
		}
	}
	Log("backend", "error", "handler", "Notification not found for delete")
	c.JSON(404, gin.H{"error": "Notification not found"})
}

func main() {
	r := gin.Default()

	r.GET("/notifications", getNotifications)
	r.GET("/notifications/:id", getNotification)
	r.POST("/notifications", createNotification)
	r.PUT("/notifications/:id", updateNotification)
	r.DELETE("/notifications/:id", deleteNotification)

	Log("backend", "info", "main", "Notification App Backend started")
	fmt.Println("Server running on port 8081")
	r.Run(":8081")
}