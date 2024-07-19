# Cascading Failures with Delays Scenario

## Overview

This scenario simulates cascading failures across multiple services in a microservices architecture, introducing delays between the failure of one service and the appearance of symptoms in dependent services. The scenario helps identify how failures propagate through the system and affect downstream services, providing insights into the resilience and robustness of the overall architecture.

## Topology

- **Inventory Service**: Manages product inventory.
- **Order Service**: Manages order processing and depends on the Inventory Service.
- **Payment Service**: Handles payment transactions and depends on the Order Service.
- **Frontend Service**: Provides the user-facing interface and depends on the Order Service and Payment Service.

## Objective

To test the resilience of the microservices architecture by inducing failures in the Inventory Service and observing the propagation of these failures to the Order Service, Payment Service, and Frontend Service, with delays introduced to simulate real-life scenarios.

## Folder Structure

```plaintext
scenarios/
├── cascading-failures/
│   ├── inventory-plan.yaml
│   ├── inventory-vs.yaml
│   ├── order-plan.yaml
│   ├── order-vs.yaml
│   ├── payment-vs.yaml
│   ├── frontend-vs.yaml
│   ├── gateway.yaml
│   ├── run.sh
│   ├── README.md
```

## Configuration Files

### inventory-plan.yaml

This file defines the ChaosMania plan for the Inventory Service.

### inventory-vs.yaml

This file defines the VirtualService for the Inventory Service.

### order-plan.yaml

This file defines the ChaosMania plan for the Order Service.

### order-vs.yaml

This file defines the VirtualService for the Order Service.

### payment-vs.yaml

This file defines the VirtualService for the Payment Service.

### frontend-vs.yaml

This file defines the VirtualService for the Frontend Service.

### gateway.yaml

This file defines the Istio Gateway configuration.

### run.sh

This script sets up the environment, deploys the necessary services, and runs the ChaosMania scenarios.

## Error Propagation Schema

```plaintext
┌──────────────────┐       ┌────────────────┐       ┌────────────────┐
│ Inventory        │       │ Order          │       │ Payment        │
│ Service          │──────▶│ Service        │──────▶│ Service        │
│ (Simulate        │       │ (High CPU,     │       │ (Memory        │
│ Memory Load)     │       │ Memory Load)   │       │ Allocation)    │
└──────────────────┘       └────────────────┘       └────────────────┘
         │                        │                        │
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
┌──────────────────┐       ┌────────────────┐       ┌────────────────┐
│ Sleep 10s        │       │ Sleep 10s      │       │ Sleep 10s      │
│ (Delay Propagation) │       │ (Delay Propagation) │       │ (Delay Propagation) │
└──────────────────┘       └────────────────┘       └────────────────┘
         │                        │                        │
         ▼                        │                        │
┌──────────────────┐              │                        │
│                  │              │                        │
│ Frontend         │◀─────────────┘                        │
│ Service          │                                       │
│ (Memory          │◀──────────────────────────────────────┘
│ Allocation,      │
│ Redis Command)   │
└──────────────────┘