package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Notification from test server
type Notification struct {
	ID        string `json:"ID"`
	Type      string `json:"Type"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
	Priority  int
}

// Priority weight based on type
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

// Min-Heap implementation
type MinHeap []Notification

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool {
	if h[i].Priority != h[j].Priority {
		return h[i].Priority < h[j].Priority
	}
	return h[i].Timestamp < h[j].Timestamp
}
func (h MinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *MinHeap) Push(x interface{}) {
	*h = append(*h, x.(Notification))
}
func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Fetch notifications from test server
func fetchNotifications() ([]Notification, error) {
	req, _ := http.NewRequest("GET",
		"http://20.207.122.201/evaluation-service/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Notifications []Notification `json:"notifications"`
	}
	json.Unmarshal(body, &result)
	return result.Notifications, nil
}

// Get top 10 priority notifications
func getTop10(notifications []Notification) []Notification {
	h := &MinHeap{}
	heap.Init(h)

	for _, n := range notifications {
		n.Priority = getWeight(n.Type)
		if h.Len() < 10 {
			heap.Push(h, n)
		} else if n.Priority > (*h)[0].Priority ||
			(n.Priority == (*h)[0].Priority && n.Timestamp > (*h)[0].Timestamp) {
			heap.Pop(h)
			heap.Push(h, n)
		}
	}

	// Convert heap to sorted slice
	result := make([]Notification, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(Notification)
	}
	return result
}

// API Handler
func getPriorityInbox(c *gin.Context) {
	Log("backend", "info", "handler", "Fetching priority inbox")

	notifications, err := fetchNotifications()
	if err != nil {
		Log("backend", "error", "handler", "Failed to fetch notifications")
		c.JSON(500, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	top10 := getTop10(notifications)
	Log("backend", "info", "handler", fmt.Sprintf("Returning top %d notifications", len(top10)))
	c.JSON(200, gin.H{
		"top_notifications": top10,
		"count":             len(top10),
	})
}