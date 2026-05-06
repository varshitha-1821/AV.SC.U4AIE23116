package main

import (
	"bytes"
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJNYXBDbGFpbXMiOnsiYXVkIjoiaHR0cDovLzIwLjI0NC41Ni4xNDQvZXZhbHVhdGlvbi1zZXJ2aWNlIiwiZW1haWwiOiJhdi5zYy51NGFpZTIzMTE2QGF2LnN0dWRlbnRzLmFtcml0YS5lZHUiLCJleHAiOjE3NzgwNjQ3MDAsImlhdCI6MTc3ODA2MzgwMCwiaXNzIjoiQWZmb3JkIE1lZGljYWwgVGVjaG5vbG9naWVzIFByaXZhdGUgTGltaXRlZCIsImp0aSI6ImE0NTFlYmIyLTU3NDItNDVlNC1hODZjLTAwZjRmYzBlODFhNyIsImxvY2FsZSI6ImVuLUlOIiwibmFtZSI6ImphanVsYSB5b2dhIHZhcnNoaXRoYSIsInN1YiI6ImQ2ZjE4NTkyLTExZDktNDQ3OC04NDlhLTkxZTY4NmY1MmMxMiJ9LCJlbWFpbCI6ImF2LnNjLnU0YWllMjMxMTZAYXYuc3R1ZGVudHMuYW1yaXRhLmVkdSIsIm5hbWUiOiJqYWp1bGEgeW9nYSB2YXJzaGl0aGEiLCJyb2xsTm8iOiJhdi5zYy51NGFpZTIzMTE2IiwiYWNjZXNzQ29kZSI6IlBUQk1tUSIsImNsaWVudElEIjoiZDZmMTg1OTItMTFkOS00NDc4LTg0OWEtOTFlNjg2ZjUyYzEyIiwiY2xpZW50U2VjcmV0IjoiaHFVbUVmeVJ6UnpHRFhRSyJ9.D92oOTRXbMzO0MWf4MHQo-rBpOqPsgfPVPQB8q6sUvA"

func Log(stack, level, pkg, message string) {
	payload := map[string]string{
		"stack": stack, "level": level,
		"package": pkg, "message": message,
	}
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "http://20.207.122.201/evaluation-service/logs", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
}

// Local notification model
type Notification struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Server notification model
type ServerNotification struct {
	ID        string `json:"ID"`
	Type      string `json:"Type"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
	Priority  int
}

var notifications []Notification
var nextID = 1

// CRUD Handlers
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

func deleteNotificationHandler(c *gin.Context) {
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

// Priority Inbox
func getWeight(notifType string) int {
	switch notifType {
	case "Placement":
		return 3
	case "Result":
		return 2
	case "Event":
		return 1
	default:
		return 0
	}
}

type MinHeap []ServerNotification

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool {
	if h[i].Priority != h[j].Priority {
		return h[i].Priority < h[j].Priority
	}
	return h[i].Timestamp < h[j].Timestamp
}
func (h MinHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x interface{}) { *h = append(*h, x.(ServerNotification)) }
func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func fetchServerNotifications() ([]ServerNotification, error) {
	req, _ := http.NewRequest("GET", "http://20.207.122.201/evaluation-service/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Notifications []ServerNotification `json:"notifications"`
	}
	json.Unmarshal(body, &result)
	return result.Notifications, nil
}

func getTopN(notifs []ServerNotification, n int) []ServerNotification {
	h := &MinHeap{}
	heap.Init(h)
	for _, notif := range notifs {
		notif.Priority = getWeight(notif.Type)
		if h.Len() < n {
			heap.Push(h, notif)
		} else if notif.Priority > (*h)[0].Priority {
			heap.Pop(h)
			heap.Push(h, notif)
		}
	}
	result := make([]ServerNotification, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(ServerNotification)
	}
	return result
}

func getPriorityInbox(c *gin.Context) {
	Log("backend", "info", "handler", "Fetching priority inbox")

	n := 10
	if nStr := c.Query("n"); nStr != "" {
		if val, err := strconv.Atoi(nStr); err == nil && val > 0 {
			n = val
		}
	}

	notifs, err := fetchServerNotifications()
	if err != nil {
		Log("backend", "error", "handler", "Failed to fetch notifications")
		c.JSON(500, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	topN := getTopN(notifs, n)
	Log("backend", "info", "handler", fmt.Sprintf("Returning top %d notifications", len(topN)))
	c.JSON(200, gin.H{"top_notifications": topN, "count": len(topN)})
}

func main() {
	r := gin.Default()

	r.GET("/notifications", getNotifications)
	r.GET("/notifications/priority", getPriorityInbox)
	r.GET("/notifications/:id", getNotification)
	r.POST("/notifications", createNotification)
	r.PUT("/notifications/:id", updateNotification)
	r.DELETE("/notifications/:id", deleteNotificationHandler)

	Log("backend", "info", "main", "Notification App Backend started")
	fmt.Println("Server running on port 8081")
	r.Run(":8081")
}