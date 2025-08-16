# Idempotent Webhook Relay

## Build

Receive Stripe test webhooks, store idempotency keys, retry with backoff to downstream “consumer” services, dead-letter UI.

## Stack

Go + Gin, outbox table, SQS/RabbitMQ, small Next.js dashboard.

## Showcase

Exactly-once semantics, backpressure, outbox pattern, tracing with OpenTelemetry.

# Architecture

## Component / Flow Diagram

```plantuml
@startuml
title Idempotent Webhook Relay — Components & Data Flow

skinparam componentStyle rectangle
skinparam wrapWidth 180
skinparam maxMessageSize 200

rectangle "External" as ext {
  component "Stripe\n(Webhooks)" as Stripe <<External>>
}

rectangle "Webhook Relay Service (Go + Gin)" as relay {
  component "Gin Webhook Receiver\n(Signature verify +\nIdempotency key extraction)" as Gin
  database  "Outbox Table\n(Postgres/MySQL)\n• events\n• idempotency_keys (UNIQUE)" as Outbox
  component "Outbox Forwarder\n(worker / cron / loop)\n• SELECT ... FOR UPDATE SKIP LOCKED\n• publish with backoff" as Forwarder
}

rectangle "Messaging" as msg {
  component "RabbitMQ or SQS\n<<Queue>>" as MQ
  component "Dead-Letter Queue\n(RabbitMQ DLX / SQS DLQ)" as DLQ
}

rectangle "Consumers" as cons {
  component "Outbox Processor\n<<Consumer>>\n• idempotent handler\n• acks/settles" as Consumer
  component "Downstream Services\n(Charge/Invoice sync, etc.)" as Downstream
}

rectangle "UI" as ui {
  component "Next.js Dashboard\n• Dead-letter viewer\n• Replay to main queue" as Dashboard
}

rectangle "Observability" as obs {
  component "OpenTelemetry SDK\n(in Relay + Consumer)" as OTelSDK
  component "OTel Collector" as OTelCollector
  component "Traces Backend\n(Jaeger/Tempo/etc.)" as Traces
}

Stripe -down-> Gin : HTTPS webhook (event)
Gin -down-> Outbox : insert(event, idempotency_key)\n(on conflict do nothing)
Forwarder -right-> Outbox : poll pending events
Forwarder -right-> MQ : publish message\n(with key/dedup id)
MQ -right-> Consumer : deliver message
Consumer -down-> Downstream : process (exactly-once)
Consumer -down-> MQ : ack / settle\n(or reject -> DLQ)
MQ -down-> DLQ : dead-letter (after retries)
Dashboard -left-> DLQ : list / inspect / replay

OTelSDK -down-> OTelCollector
OTelCollector -down-> Traces

note right of Outbox
  Outbox pattern ensures durable handoff.
  UNIQUE(idempotency_key) prevents duplicates
  even if Stripe retries.
end note

note bottom of MQ
  Backpressure via bounded queue & consumer concurrency.
end note

@enduml
```

## Sequence — “The Good”: exactly-once happy path

```plantuml
@startuml
title Good Path — Exactly-Once Relay

autonumber
skinparam maxMessageSize 200
actor Stripe
participant "Gin Webhook Receiver\n(Go + Gin)" as Gin
database  "Outbox Table\n(idempotency_keys UNIQUE)" as Outbox
participant "Outbox Forwarder\n(worker)" as Fwd
participant "Queue\n(RabbitMQ/SQS)" as MQ
participant "Outbox Processor\n(Consumer)" as C
participant "Downstream Service(s)" as Svc
participant "OTel SDK" as OTel

Stripe -> Gin : POST /webhook (event, headers)\nStripe-Signature + Idempotency-Key
activate Gin
Gin -> OTel : start span 'webhook_received'
Gin -> Gin : verify signature
Gin -> Outbox : INSERT event + idempotency_key\nON CONFLICT DO NOTHING
Outbox --> Gin : OK (new) or ignored (duplicate)
deactivate Gin

... async polling ...
Fwd -> Outbox : SELECT pending FOR UPDATE SKIP LOCKED
Outbox --> Fwd : event rows
Fwd -> MQ : Publish(event, dedup key)
MQ --> Fwd : accepted

MQ -> C : Deliver message
activate C
C -> OTel : start span 'process_event'
C -> Svc : Handle event (idempotent)
Svc --> C : OK
C -> MQ : ACK
deactivate C

OTel -> OTel : export spans -> Collector/Backend

@enduml
```

## Sequence — “The Bad”: retries, backoff, and dead-letter UI

```plantuml
@startuml
title Bad Path — Retry with Backoff → DLQ → Manual Replay

autonumber
skinparam maxMessageSize 220
actor Stripe
participant "Gin Webhook Receiver" as Gin
database  "Outbox Table" as Outbox
participant "Outbox Forwarder" as Fwd
participant "Queue\n(RabbitMQ/SQS)" as MQ
participant "Outbox Processor\n(Consumer)" as C
participant "Dead-Letter Queue" as DLQ
actor "Next.js Dashboard\n(User)" as UI

Stripe -> Gin : POST /webhook (event)
Gin -> Outbox : INSERT event (unique key)
... async ...
Fwd -> Outbox : fetch pending
Fwd -> MQ : publish(event)
MQ -> C : deliver
activate C
C -> C : process fails (transient/bug)
C --> MQ : NACK / reject
deactivate C

loop Retry with exponential backoff (1x, 2x, 4x, ...)
  MQ -> C : redeliver
  activate C
  C -> C : still fails
  C --> MQ : NACK
  deactivate C
end

MQ -> DLQ : move message (max retries exceeded)

== Dead-letter handling ==
UI -> DLQ : list/inspect failed messages
UI -> DLQ : replay -> main queue
DLQ --> MQ : requeue copy
MQ -> C : deliver again (after fix)
C -> C : succeeds; ACK

@enduml
```
