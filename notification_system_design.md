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
