package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ========================
// DATA STRUCTURES
// ========================

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	IsRead    bool      `json:"is_read"`
}

// Raw response from the external evaluation API
type APINotification struct {
	ID        string `json:"ID"`
	Type      string `json:"Type"`
	Message   string `json:"Message"`
	Timestamp string `json:"Timestamp"`
}

type NotificationsAPIResponse struct {
	Notifications []APINotification `json:"notifications"`
}

// ========================
// PRIORITY QUEUE (MAX HEAP)
// For Stage 6: Priority Inbox
// Priority: Placement(3) > Result(2) > Event(1), tiebreak by recency
// ========================

var typeWeight = map[string]int{
	"Placement": 3,
	"Result":    2,
	"Event":     1,
}

type PQItem struct {
	notif    Notification
	priority int
	index    int
}

type PriorityQueue []*PQItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority // max-heap
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PQItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

// computePriority: type weight * large constant + unix timestamp
// This ensures Placement always beats Result, which always beats Event
// Within the same type, more recent timestamp wins
func computePriority(n Notification) int {
	weight := typeWeight[n.Type]
	return weight*1_000_000_000_000 + int(n.Timestamp.Unix())
}

// getTopN returns the top N notifications by priority using the heap
func getTopN(notifications []Notification, n int) []Notification {
	pq := make(PriorityQueue, 0, len(notifications))
	heap.Init(&pq)

	for _, notif := range notifications {
		item := &PQItem{
			notif:    notif,
			priority: computePriority(notif),
		}
		heap.Push(&pq, item)
	}

	result := make([]Notification, 0, n)
	for i := 0; i < n && pq.Len() > 0; i++ {
		item := heap.Pop(&pq).(*PQItem)
		result = append(result, item.notif)
	}
	return result
}

// ========================
// LOGGING MIDDLEWARE
// ========================

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		log.Printf("[REQUEST] Method=%s | Path=%s | IP=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
		)
		c.Next()
		log.Printf("[RESPONSE] Method=%s | Path=%s | Status=%d | Duration=%v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

// ========================
// FETCH FROM EXTERNAL API
// ========================

func fetchNotificationsFromAPI() ([]Notification, error) {
	baseURL := os.Getenv("EVAL_API_BASE")
	if baseURL == "" {
		baseURL = "http://20.207.122.201"
	}
	url := baseURL + "/evaluation-service/notifications"

	apiToken := os.Getenv("API_TOKEN")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[ERROR] Creating request for %s: %v", url, err)
		return nil, err
	}

	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] HTTP GET %s failed: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Reading response body: %v", err)
		return nil, err
	}

	log.Printf("[FETCH] URL=%s | StatusCode=%d | BodySize=%d bytes", url, resp.StatusCode, len(body))

	var apiResp NotificationsAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("[ERROR] JSON unmarshal failed: %v", err)
		return nil, err
	}

	// Convert API format to internal Notification format
	notifications := make([]Notification, 0, len(apiResp.Notifications))
	for _, a := range apiResp.Notifications {
		// Parse timestamp — handle both formats
		ts, err := time.Parse("2006-01-02 15:04:05", a.Timestamp)
		if err != nil {
			ts, err = time.Parse(time.RFC3339, a.Timestamp)
			if err != nil {
				log.Printf("[WARN] Could not parse timestamp '%s', using now", a.Timestamp)
				ts = time.Now()
			}
		}
		notifications = append(notifications, Notification{
			ID:        a.ID,
			Type:      a.Type,
			Message:   a.Message,
			Timestamp: ts,
			IsRead:    false,
		})
	}

	log.Printf("[FETCH] Parsed %d notifications from API", len(notifications))
	return notifications, nil
}

// ========================
// HANDLERS
// ========================

// GET /health
func healthHandler(c *gin.Context) {
	log.Printf("[HEALTH] Health check called")
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "notification-app-be",
		"port":    "8081",
	})
}

// GET /notifications — returns ALL notifications
func getAllNotificationsHandler(c *gin.Context) {
	log.Printf("[HANDLER] getAllNotifications called")

	notifications, err := fetchNotificationsFromAPI()
	if err != nil {
		log.Printf("[ERROR] getAllNotifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[HANDLER] Returning %d notifications", len(notifications))
	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// GET /notifications/priority?n=10 — returns top N by priority (Stage 6)
func getPriorityNotificationsHandler(c *gin.Context) {
	nStr := c.DefaultQuery("n", "10")
	n := 10
	fmt.Sscanf(nStr, "%d", &n)
	if n <= 0 {
		n = 10
	}

	log.Printf("[HANDLER] getPriorityNotifications called with n=%d", n)

	notifications, err := fetchNotificationsFromAPI()
	if err != nil {
		log.Printf("[ERROR] getPriorityNotifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	top := getTopN(notifications, n)

	log.Printf("[HANDLER] Returning top %d priority notifications out of %d total", len(top), len(notifications))
	c.JSON(http.StatusOK, gin.H{
		"top_notifications": top,
		"count":             len(top),
		"priority_order":    "Placement > Result > Event, tiebreak by most recent timestamp",
	})
}

// POST /notifications — create a new notification
func createNotificationHandler(c *gin.Context) {
	var input struct {
		UserID  int    `json:"user_id" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[ERROR] createNotification bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notif := Notification{
		ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
		Type:      input.Type,
		Message:   input.Message,
		Timestamp: time.Now(),
		IsRead:    false,
	}

	log.Printf("[CREATE] Notification created | Type=%s | UserID=%d | Message=%s",
		notif.Type, input.UserID, notif.Message)

	c.JSON(http.StatusCreated, gin.H{
		"id":         notif.ID,
		"user_id":    input.UserID,
		"type":       notif.Type,
		"message":    notif.Message,
		"status":     "pending",
		"created_at": notif.Timestamp.Format("2006-01-02 15:04:05"),
	})
}

// PUT /notifications/:id — update a notification (mark as read, etc.)
func updateNotificationHandler(c *gin.Context) {
	id := c.Param("id")

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[ERROR] updateNotification bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[UPDATE] Notification id=%s | Updates=%v", id, input)

	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"message":    "Notification updated",
		"updates":    input,
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// DELETE /notifications/:id — delete a notification
func deleteNotificationHandler(c *gin.Context) {
	id := c.Param("id")

	log.Printf("[DELETE] Notification id=%s", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification deleted",
		"id":      id,
	})
}

// ========================
// MAIN
// ========================

func main() {
	r := gin.New()
	r.Use(LoggerMiddleware())
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", healthHandler)

	// Notification routes
	r.GET("/notifications", getAllNotificationsHandler)
	r.GET("/notifications/priority", getPriorityNotificationsHandler)
	r.POST("/notifications", createNotificationHandler)
	r.PUT("/notifications/:id", updateNotificationHandler)
	r.DELETE("/notifications/:id", deleteNotificationHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Printf("[STARTUP] Notification App BE running on port %s\n", port)
	log.Printf("[STARTUP] Server starting on :%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[FATAL] Server failed to start: %v", err)
	}
}