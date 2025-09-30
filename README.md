# Idempotent Webhook Relay

[![Go](https://img.shields.io/badge/Go-1.24-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Powered-blue.svg)](https://www.docker.com/)

An idempotent webhook relay service designed to receive Stripe webhooks, ensure exactly-once processing, and reliably forward events to downstream services. This project implements the **Outbox Pattern** to guarantee data consistency and resilience, even in the face of network failures or service outages.

## Overview

Processing webhooks from external services like Stripe requires careful handling to avoid duplicate processing, which could lead to incorrect billing, multiple notifications, or inconsistent data. This service solves that problem by providing a robust, fault-tolerant relay system with three decoupled microservices.

The core architecture consists of three specialized microservices:

1.  **Webhook Service**: A public-facing API that ingests incoming webhooks from Stripe, verifies their signature, and stores them in a database.
2.  **Producer Service**: A background worker that polls the database for new events and publishes them to a message queue.
3.  **Consumer Service**: A concurrent worker that processes events from the queue with exponential backoff retry logic.

This separation of concerns ensures that webhook ingestion is fast and reliable, while the actual processing can be scaled and managed independently.

## Features

-   **Idempotent Ingestion**: Prevents duplicate webhook processing using a `UNIQUE` constraint on the Stripe event ID in the database.
-   **Outbox Pattern**: Atomically saves incoming events to a PostgreSQL `outbox` table before acknowledging them, ensuring no events are lost.
-   **Guaranteed Delivery**: Uses RabbitMQ as a message broker to ensure at-least-once delivery of events to downstream consumers.
-   **Stripe Integration**: Includes built-in signature verification for authenticating Stripe webhooks.
-   **Concurrent Processing**: Consumer service processes multiple events concurrently with a configurable worker pool.
-   **Exponential Backoff**: Implements sophisticated retry logic with jitter for failed event processing.
-   **Decoupled & Scalable**: Microservice architecture allows each component (ingestion, production, consumption) to be scaled independently.
-   **Enhanced Outbox Schema**: Rich event tracking with status, retry counts, error logging, and provider information.
-   **Type-Safe SQL**: Uses `sqlc` for generating type-safe Go code from raw SQL queries, preventing runtime SQL errors.
-   **Embedded Migrations**: Database migrations are managed by `goose` and embedded directly into the application binary.
-   **Containerized**: Fully configured with Docker and Docker Compose for easy setup and consistent environments.
-   **Clean Architecture**: Follows clean architecture principles with distinct layers for handlers, business logic, and data access.

## Architecture Flow

1.  **Stripe** sends a webhook event to the `webhook` service's public endpoint (`/stripe/webhook`).
2.  The **`webhook` service** verifies the `Stripe-Signature` header.
3.  Upon successful verification, it inserts the event into a PostgreSQL `outbox` table. A `UNIQUE` constraint on the `event_id` column ensures that duplicate webhooks from Stripe are ignored, achieving idempotency.
4.  The service immediately returns a `200 OK` response to Stripe.
5.  The **`producer` service** periodically polls the `outbox` table for unprocessed events.
6.  For each new event, the `producer` publishes it as a message to a **RabbitMQ** queue and marks the event as `pending`.
7.  The **`consumer` service** listens to the RabbitMQ queue with a pool of concurrent workers, processes messages with exponential backoff retry logic, and updates the outbox status upon completion (`processed`) or failure (`process_failed`).

![Architecture Diagram](https://raw.githubusercontent.com/petechu/idempotent-webhook-relay/main/overview.png)
*(A similar diagram is available in PlantUML format in `overview.md`)*

## Tech Stack

-   **Backend**: Go 1.24
-   **Web Framework**: Gin
-   **Database**: PostgreSQL
-   **Message Queue**: RabbitMQ
-   **Database Tooling**:
    -   `sqlc` for type-safe SQL query generation.
    -   `pgx/v5` as the PostgreSQL driver.
    -   `goose` for database migrations.
-   **Containerization**: Docker & Docker Compose
-   **Development Tools**:
    -   `air` for hot-reloading during development.
    -   `make` for convenient command execution.

## Getting Started

Follow these instructions to get the project up and running on your local machine.

### Prerequisites

-   [Go](https://go.dev/doc/install) (version 1.24 or later)
-   [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
-   [Make](https://www.gnu.org/software/make/)

### Installation

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/petechu/idempotent-webhook-relay.git
    cd idempotent-webhook-relay
    ```

2.  **Set up environment variables:**
    The application is configured using environment variables. Create a `.env` file in the project root:

    ```bash
    touch .env
    ```

    Open the `.env` file and add your Stripe API keys. The database variables have defaults but can be overridden here as well.

    ```dotenv
    # .env
    STRIPE_SECRET_KEY=sk_test_...
    STRIPE_WEBHOOK_SECRET=whsec_...

    # Optional: Override database defaults
    # DB_HOST=localhost
    # DB_PORT=5432
    # DB_NAME=idempotent-webhook-relay
    # DB_USERNAME=postgres
    # DB_PASSWORD=postgres
    ```

## Usage

You can run the application stack using Docker Compose or run the services locally for development.

### Running with Docker Compose (Recommended)

This method builds the Go application and starts containers for the webhook service, PostgreSQL, and RabbitMQ.

1.  **Start all services:**

    ```bash
    docker-compose up --build -d
    ```

    The webhook service will be available at `http://localhost:3000`.

    **Note**: The default `docker-compose.yml` only runs the `webhook` service. To run the full producer/consumer flow, either extend the file to include services for `producer` and `consumer` or use the local development method below.

2.  **Check logs:**
    To view the logs for a specific service (e.g., `webhook`):

    ```bash
    docker-compose logs -f webhook
    ```

3.  **Stop all services:**
    To stop and remove the containers, network, and volumes:

    ```bash
    docker-compose down
    ```

### Running Locally for Development (Full Stack)

This method allows you to run the complete three-service architecture on your host machine while using Docker for dependencies.

1.  **Start dependencies (PostgreSQL & RabbitMQ):**

    ```bash
    docker-compose up -d db rabbitmq
    ```

    -   The RabbitMQ management UI will be available at `http://localhost:15672` (login with `guest` / `guest`).

2.  **Run database migrations:**
    This command applies all pending database migrations defined in `internal/db/migrations`.

    ```bash
    make up
    ```

3.  **Run the services:**
    Open three separate terminal windows and run each service. Ensure your `.env` file is present in the project root.
    -   **Terminal 1: Webhook Service**

        ```bash
        go run cmd/webhook/main.go
        ```
        *Listens for incoming webhooks at `http://localhost:3000`.*

    -   **Terminal 2: Producer Service**

        ```bash
        go run cmd/producer/main.go
        ```
        *Polls the database and publishes events to RabbitMQ.*

    -   **Terminal 3: Consumer Service**

        ```bash
        go run cmd/consumer/main.go
        ```
        *Consumes events from RabbitMQ and processes them.*

4.  **(Optional) Hot Reloading with Air:**
    If you have [Air](https://github.com/cosmtrek/air) installed, you can run the webhook service with hot-reloading enabled.

    ```bash
    air
    ```

## Database Migrations

Database schema changes are managed using `goose`. The `Makefile` provides convenient commands.

-   **Apply all pending migrations:**

    ```bash
    make up
    ```

-   **Check the status of migrations:**

    ```bash
    make db-status
    ```

-   **Roll back all migrations (Warning: destructive operation):**

    ```bash
    make reset
    ```

## Project Structure

```
├── cmd/                    # Application entry points for each service
│   ├── consumer/           # Event processor service (consumes from RabbitMQ)
│   ├── producer/           # Event publisher service (polls DB, sends to RabbitMQ)
│   └── webhook/            # HTTP webhook receiver service
├── internal/
│   ├── config/             # Environment variable configuration
│   ├── db/                 # Database models, migrations, and sqlc-generated code
│   │   ├── migrations/     # SQL schema migrations (embedded with goose)
│   │   └── query.sql.go    # sqlc-generated type-safe Go code
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
├── query.sql               # SQL queries for sqlc to generate Go code from
├── overview.md             # Architecture diagrams in PlantUML
└── README.md               # This file
```

## Roadmap

-   **Enhanced Consumer Logic**: Implement meaningful business logic for processing different types of Stripe events.
-   **Dead-Letter Queue (DLQ)**: Configure RabbitMQ's Dead Letter Exchange (DLX) to handle messages that fail processing after all retries.
-   **Observability**: Add structured logging, metrics (Prometheus), and distributed tracing (OpenTelemetry).
-   **Admin Dashboard**: A simple UI (e.g., Next.js) to view the status of events, inspect payloads, and manually replay events from the DLQ.
-   **Multi-Provider Support**: Extend the system to handle webhooks from other providers beyond Stripe.

## License

This project is licensed under the MIT License.
