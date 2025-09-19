# Idempotent Webhook Relay

[![Go](https://img.shields.io/badge/Go-1.24-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Powered-blue.svg)](https://www.docker.com/)

An idempotent webhook relay service designed to receive Stripe webhooks, ensure exactly-once processing, and reliably forward events to downstream services. This project implements the **Outbox Pattern** to guarantee data consistency and resilience, even in the face of network failures or service outages.

## Overview

Processing webhooks from external services like Stripe requires careful handling to avoid duplicate processing, which could lead to incorrect billing, multiple notifications, or inconsistent data. This service solves that problem by providing a robust, fault-tolerant relay system.

The core architecture consists of three decoupled microservices:

1. **Webhook Service**: A public-facing API that ingests incoming webhooks from Stripe.
2. **Producer Service**: A background worker that polls the database for new events and publishes them to a message queue.
3. **Consumer Service**: A worker that processes events from the queue and executes the required business logic.

This separation of concerns ensures that webhook ingestion is fast and reliable, while the actual processing can be scaled and managed independently.

## Features

- **Idempotent Ingestion**: Prevents duplicate webhook processing using unique constraints on the Stripe event ID in the database.
- **Outbox Pattern**: Atomically saves incoming events to a PostgreSQL `outbox` table before acknowledging them, ensuring no events are lost.
- **Guaranteed Delivery**: Uses RabbitMQ as a message broker to ensure at-least-once delivery of events to downstream consumers.
- **Stripe Integration**: Includes built-in signature verification for authenticating Stripe webhooks.
- **Decoupled & Scalable**: Microservice architecture allows each component (ingestion, production, consumption) to be scaled independently.
- **Containerized**: Fully configured with Docker and Docker Compose for easy setup and consistent environments.
- **Database Migrations**: Manages database schema with `goose`, with migrations embedded directly into the application binary.
- **Clean Architecture**: Follows clean architecture principles with distinct layers for handlers, business logic, and data access.

## Architecture Flow

1. **Stripe** sends a webhook event to the `webhook` service's public endpoint (`/stripe/webhook`).
2. The **`webhook` service** verifies the `Stripe-Signature` header.
3. Upon successful verification, it attempts to insert the event into a PostgreSQL `outbox` table. A `UNIQUE` constraint on the `event_id` column ensures that duplicate webhooks from Stripe are ignored, achieving idempotency.
4. The service immediately returns a `200 OK` response to Stripe.
5. The **`producer` service** periodically polls the `outbox` table for unprocessed events.
6. For each new event, the `producer` publishes it as a message to a **RabbitMQ** queue and marks the event as `pending`.
7. The **`consumer` service** listens to the RabbitMQ queue, receives messages, and performs the final processing. (Note: The consumer logic is currently a placeholder).

![Architecture Diagram](https://raw.githubusercontent.com/petechu/idempotent-webhook-relay/main/overview.png)
_(Diagram generated from `overview.md` PlantUML)_

## Tech Stack

- **Backend**: Go
- **Web Framework**: Gin
- **Database**: PostgreSQL
- **Message Queue**: RabbitMQ
- **Database Tooling**:
  - `sqlc` for type-safe SQL query generation.
  - `pgx/v5` as the PostgreSQL driver.
  - `goose` for database migrations.
- **Containerization**: Docker & Docker Compose

## Getting Started

Follow these instructions to get the project up and running on your local machine.

### Prerequisites

- [Go](https://go.dev/doc/install) (version 1.24 or later)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

### Installation

1. **Clone the repository:**

    ```bash
    git clone https://github.com/petechu/idempotent-webhook-relay.git
    cd idempotent-webhook-relay
    ```

2. **Set up environment variables:**
    The application is configured using environment variables. Create a `.env` file in the root of the project by copying the example:

    ```bash
    cp .env.example .env
    ```

    Now, open the `.env` file and fill in your Stripe API keys:

    ```dotenv
    # .env
    STRIPE_SECRET_KEY=sk_test_...
    STRIPE_WEBHOOK_SECRET=whsec_...
    ```

### Usage

The recommended way to run the entire application stack is with Docker Compose.

#### Running with Docker (Recommended)

This command will build the Go application, start containers for PostgreSQL and RabbitMQ, and run all services.

1. **Start all services:**

    ```bash
    docker-compose up --build -d
    ```

    The webhook service will be available at `http://localhost:3000`.

2. **Check logs:**
    To view the logs for a specific service (e.g., `webhook`):

    ```bash
    docker-compose logs -f webhook
    ```

3. **Stop all services:**
    To stop and remove the containers, network, and volumes:

    ```bash
    docker-compose down
    ```

#### Running Locally for Development

If you prefer to run the Go services directly on your host machine for development, you can use Docker Compose to manage only the external dependencies.

1. **Start dependencies (PostgreSQL & RabbitMQ):**

    ```bash
    docker-compose up -d db rabbitmq
    ```

2. **Run database migrations:**
    This command applies all pending database migrations defined in `internal/db/migrations`.

    ```bash
    make up
    ```

3. **Run the services:**
    Open three separate terminal windows and run each service. Ensure your `.env` file is present.
    - **Terminal 1: Webhook Service**

      ```bash
      go run cmd/webhook/main.go
      ```

    - **Terminal 2: Producer Service**

      ```bash
      go run cmd/producer/main.go
      ```

    - **Terminal 3: Consumer Service**

      ```bash
      go run cmd/consumer/main.go
      ```

4. **(Optional) Hot Reloading with Air:**
    If you have [Air](https://github.com/cosmtrek/air) installed, you can run the webhook service with hot-reloading enabled.

    ```bash
    air
    ```

## Database Migrations

Database schema changes are managed using `goose`. The `Makefile` provides convenient commands.

- **Apply all pending migrations:**

  ```bash
  make up
  ```

- **Check the status of migrations:**

  ```bash
  make db-status
  ```

- **Roll back all migrations (Warning: this is destructive and will delete all data):**

  ```bash
  make reset
  ```

## Project Structure

```
├── cmd/                    # Application entry points for each service
│   ├── webhook/            # HTTP webhook receiver service
│   ├── producer/           # Event publisher service (polls DB and sends to RabbitMQ)
│   └── consumer/           # Event processor service (consumes from RabbitMQ)
├── internal/
│   ├── config/             # Environment variable configuration
│   ├── db/                 # Database models, migrations, and sqlc-generated code
│   ├── handler/            # HTTP handlers, routes, and middleware
│   ├── logic/              # Core business logic
│   ├── queue/              # RabbitMQ abstraction layer
│   ├── svc/                # Service context for dependency injection
│   └── utils/              # Shared helper functions
├── .air.toml               # Configuration for live-reloading with Air
├── docker-compose.yml      # Defines the multi-container application stack
├── Dockerfile              # Docker build instructions for the Go application
├── go.mod                  # Go module dependencies
├── Makefile                # Helper commands for development (e.g., migrations)
└── query.sql               # SQL queries for sqlc to generate Go code from
```

## Roadmap

The current implementation provides a solid foundation. Future enhancements include:

- **Consumer Logic**: Implement meaningful event processing in the consumer service.
- **Dead-Letter Queue (DLQ)**: Configure RabbitMQ's DLX to handle messages that fail processing after several retries.
- **Retry with Exponential Backoff**: Implement a sophisticated retry mechanism for publishing and processing events.
- **Observability**: Add structured logging, metrics (Prometheus), and distributed tracing (OpenTelemetry).
- **Admin Dashboard**: A simple UI (e.g., Next.js) to view the status of events, inspect payloads, and manually replay events from the DLQ.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
