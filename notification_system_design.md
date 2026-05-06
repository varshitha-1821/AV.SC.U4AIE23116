## Overview
A scalable notification system that sends alerts to users 
via Email, SMS, and Push Notifications.

## Architecture
- **API Layer** - REST API built with Go/Gin
- **Queue** - Message queue to handle high volume
- **Workers** - Process and send notifications
- **Database** - Store notification history

## API Endpoints
- POST /notifications - Send a notification
- GET /notifications - Get all notifications
- GET /notifications/:id - Get one notification
- DELETE /notifications/:id - Delete a notification

## Notification Types
- Email
- SMS  
- Push Notification

## Flow
1. Client sends request to API
2. API validates and pushes to queue
3. Worker picks up from queue
4. Worker sends notification
5. Status updated in database

## Database Schema
notifications table:
1. id
2. user_id
3. type (email/sms/push)
4. message
5. status (pending/sent/failed)
6. created_at

## Tech Stack
- Backend: Go with Gin framework
- Queue: Redis
- Database: PostgreSQL



# Notification System Design

## Stage 1

REST API endpoints for the notification platform:

GET /notifications
Headers: { "Authorization": "Bearer <token>", "Content-Type": "application/json" }
Response: { "notifications": [{ "id": "uuid", "studentId": "uuid", "type": "Placement/Event/Result", "message": "string", "isRead": false, "createdAt": "timestamp" }] }

GET /notifications/:id
Headers: { "Authorization": "Bearer <token>", "Content-Type": "application/json" }
Response: { "id": "uuid", "studentId": "uuid", "type": "Placement", "message": "string", "isRead": false, "createdAt": "timestamp" }

POST /notifications
Headers: { "Authorization": "Bearer <token>", "Content-Type": "application/json" }
Request: { "studentId": "uuid", "type": "Placement/Event/Result", "message": "string" }
Response: { "id": "uuid", "message": "Notification created", "createdAt": "timestamp" }

PATCH /notifications/:id/read
Headers: { "Authorization": "Bearer <token>" }
Response: { "message": "Notification marked as read" }

DELETE /notifications/:id
Headers: { "Authorization": "Bearer <token>" }
Response: { "message": "Notification deleted" }

GET /notifications/unread
Headers: { "Authorization": "Bearer <token>" }
Response: { "notifications": [...], "count": 5 }

Real-time: We use WebSockets so students get notified instantly without refreshing.

## Stage 2

We use PostgreSQL because it handles large data well and supports complex queries.

CREATE TABLE students (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  studentID UUID REFERENCES students(id),
  notificationType VARCHAR(20) CHECK (notificationType IN ('Placement','Event','Result')),
  message TEXT NOT NULL,
  isRead BOOLEAN DEFAULT false,
  createdAt TIMESTAMP DEFAULT NOW()
);

As data grows queries slow down. We fix this with indexes, pagination and archiving old data.

SELECT * FROM notifications WHERE studentID = 'uuid' ORDER BY createdAt DESC;
SELECT * FROM notifications WHERE studentID = 'uuid' AND isRead = false ORDER BY createdAt DESC;

## Stage 3

The query is slow because it scans all 5 million rows and SELECT * fetches unnecessary columns.

Fix:
CREATE INDEX idx_notifications ON notifications(studentID, isRead, createdAt DESC);
SELECT id, message, notificationType, createdAt FROM notifications
WHERE studentID = 1042 AND isRead = false ORDER BY createdAt DESC;

Adding indexes on every column is bad — it slows down inserts and wastes storage.
Only index columns you actually search by.

Students with Placement notifications in last 7 days:
SELECT DISTINCT studentID FROM notifications
WHERE notificationType = 'Placement'
AND createdAt >= NOW() - INTERVAL '7 days';

## Stage 4

Problem: DB is hit on every page load for every student which overwhelms it.

Solution 1 - Redis Cache: Check cache first, fetch DB only on miss, cache expires in 5 mins.
Tradeoff: data can be slightly stale.

Solution 2 - Pagination: Load only 20 notifications at a time.
Tradeoff: multiple requests needed.

Solution 3 - WebSocket Push: Push notifications instead of fetching on load.
Tradeoff: complex to implement.

Best approach: Redis caching + pagination together.

## Stage 5

Problems with current code: sequential loop is slow for 50,000 students,
no retry if email fails, no logging of failures, email and DB happen together.

When send_email failed for 200 students, those students never got notified
and there is no record of who failed.

Better pseudocode:

function notify_all(student_ids, message):
  batches = split(student_ids, 500)
  for batch in batches:
    parallel_for student_id in batch:
      try:
        save_to_db(student_id, message)
        send_email(student_id, message)
        push_to_app(student_id, message)
      catch error:
        add_to_retry_queue(student_id)
        log_error(student_id, error)
  process_retry_queue()

Save to DB first always so even if email fails we have a record and can retry.

## Stage 6

Priority Inbox shows top n notifications where n is chosen by user: 10, 15, 20 etc.

Priority weights: Placement = 3, Result = 2, Event = 1.
Among same type, newer timestamp wins.

We use a Min-Heap of size n. When a new notification arrives,
if it beats the lowest priority in the heap, it replaces it.
This gives O(log n) per insert instead of re-sorting everything.

See notification_app_be/main.go for working code.