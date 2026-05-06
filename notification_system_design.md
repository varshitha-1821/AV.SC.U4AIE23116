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

We need these API endpoints so the frontend can show notifications to students:

- GET /notifications → fetch all notifications for a student
- GET /notifications/:id → fetch one specific notification
- POST /notifications → create a new notification
- PATCH /notifications/:id/read → mark a notification as read
- DELETE /notifications/:id → delete a notification
- GET /notifications/unread → get only unread notifications

Each notification looks like this:
{
  "id": "unique-id",
  "studentId": "student-unique-id",
  "type": "Placement / Event / Result",
  "message": "You have a new placement opportunity",
  "isRead": false,
  "createdAt": "2026-05-06 10:00:00"
}

For real-time notifications we use WebSockets,
so students get notified instantly without refreshing the page.

---

## Stage 2

We use PostgreSQL as our database because it is reliable,
handles lots of data well, and supports complex queries easily.

Tables we need:

students table → stores student info (id, name, email)
notifications table → stores all notifications (id, studentId, type, message, isRead, createdAt)

As data grows, queries will get slower. We fix this by adding
indexes on studentId and createdAt so the database finds
data faster without scanning every single row.

---

## Stage 3

The slow query is:
SELECT * FROM notifications WHERE studentID = 1042 AND isRead = false ORDER BY createdAt DESC;

Why is it slow? Because the database is checking all 5 million rows one by one.
SELECT * also fetches columns we don't even need.

Fix → add an index and select only needed columns:
CREATE INDEX idx_notifications ON notifications(studentID, isRead, createdAt DESC);
SELECT id, message, notificationType, createdAt FROM notifications WHERE studentID = 1042 AND isRead = false ORDER BY createdAt DESC;

Should we add indexes on every column? No.
Each index makes inserts and updates slower and wastes storage.
Only add indexes on columns you actually search by.

Students who got Placement notifications in last 7 days:
SELECT DISTINCT studentID FROM notifications WHERE notificationType = 'Placement' AND createdAt >= NOW() - INTERVAL '7 days';

---

## Stage 4

Problem → the database is being hit every single time any student opens the app. With 50,000 students this kills the database.

Solution → use Redis as a cache. Think of Redis like a super fast notepad.
When a student opens the app, we check the notepad first.
If the answer is there, we return it instantly without touching the database.
If not, we fetch from database, write it to the notepad, and return it.
The notepad clears itself every 5 minutes so data stays fresh.

We also add pagination so we never load all notifications at once,
only 20 at a time.

---

## Stage 5

Problem with current code → it sends emails to 50,000 students
one by one in a loop. If it fails at student 200, the remaining
49,800 students never get notified and we don't even know who was missed.

Better approach:
- Save to database first, then send email. This way even if email fails, we have a record.
- Split 50,000 students into small batches of 500 and process them together.
- If any student fails, add them to a retry list and try again later.
- Log every failure so we always know exactly who didn't get the notification.

---

## Stage 6

Priority Inbox shows the top 10 most important unread notifications first.

Priority is decided by:
- Placement notifications → highest priority
- Result notifications → medium priority  
- Event notifications → lowest priority
- Among same type, newer ones come first

We use a Min-Heap of size 10 to keep this efficient.
As new notifications come in, we compare and swap so the
top 10 list always stays updated without re-sorting everything.

See priority_inbox.go in notification_app_be folder for the code.