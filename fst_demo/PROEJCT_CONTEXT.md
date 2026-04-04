# Project Context (Go + RabbitMQ Notification System)

We are building a small notification system using Go and RabbitMQ with a hybrid publish-subscribe model.

## Architecture

- Internal system produces events (e.g., house created/updated)
- Events are **sanitized** before publishing:
  - remove user/private data
  - keep only safe/public fields
- RabbitMQ is used for routing:
  - **Topic exchange (optional)** for event type (e.g., `listing.created`)
  - **Header exchange** for filtering (city, region, country, topology, `price_bucket`)
- Workers consume from queues and send notifications (currently just print to console)

---

## Services

### 1. API Service (`api/main.go`)

Responsibilities:
- Serves HTML at `/`
- Provides REST endpoints:
  - `GET /houses`
  - `POST /create-house`
  - `DELETE /houses/{id}`
  - `GET /filters`
  - `POST /create-filter`
- Stores data in-memory (for now)
- On house creation → publishes event to RabbitMQ
- On filter creation → creates queue + binding in RabbitMQ

---

### 2. Worker Service (`emailer/worker.go`)

Responsibilities:
- Consumes messages from RabbitMQ queues
- Sends notifications (console output for now)
- Designed to scale horizontally (multiple workers consuming same queues)

---

## Data Model

### House
- `id`
- `price`
- `price_bucket`
- `location`:
  - `city`
  - `region`
  - `country`
- `topology` (e.g., T2)
- `description`

### Filter
- `location` (same structure as House)
- `topology`
- `price_bucket`

---

## Key Constraints

- RabbitMQ **headers exchange uses exact matching**
  → numeric ranges are not supported directly  
  → use **price buckets** instead

- No sensitive data is published to RabbitMQ:
  - apply **data minimization**
  - use **pseudonymization**
  - mapping `listing_id → user` remains internal

- System prioritizes:
  - simplicity
  - scalability
  - privacy-aware design

- Full anonymity techniques (e.g., Bloom filters) are **out of scope**

---

## Tech Stack

- Go (`net/http`, possibly Gin later)
- RabbitMQ (already deployed)

---

## Goal

Build a simple, scalable, and privacy-aware notification system with clear separation between:

- internal data (private)
- messaging layer (sanitized events)
- delivery layer (workers → clients)
