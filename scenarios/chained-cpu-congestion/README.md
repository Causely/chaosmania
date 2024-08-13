# Payment Service CPU Overload Scenario

## Overview

This scenario simulates high CPU load and memory leaks in the Payment Service, causing cascading delays and performance issues in dependent services such as the Order Service and Frontend Service. The goal is to identify how high CPU usage and memory leaks in the Payment Service impact the overall system, including order processing delays and degraded user experience on the frontend.

## Topology

- **Order Service**: Manages order processing and depends on the Inventory Service.
- **Payment Service**: Handles payment transactions and depends on the Order Service.
- **Frontend Service**: Provides the user-facing interface and depends on the Order Service and Payment Service.

## Folder Structure

```plaintext
scenarios/
├── chained-cpu-congestion/
│   ├── plan.yaml
│   ├── run.sh
│   ├── README.md
```

## Configuration Files

- **plan.yaml**

This file defines the ChaosMania plan for inducing high CPU load in the Payment Service. The plan includes actions to simulate inefficient algorithms that cause the CPU load to increase progressively over time .

- **run.sh**

This script sets up the environment, deploys the necessary services, and runs the ChaosMania scenario defined in the plan.yaml file.

## Data Flow Schema

```plaintext

┌───────────────────┐
│ Client            │
└───────────────────┘
         │
         ▼
┌───────────────────┐
│ Frontend Service  │
└───────────────────┘
         │
         ▼
┌───────────────────┐
│ Order Service     │
│ (Delayed Orders)  │
└───────────────────┘
         │
         ▼
┌───────────────────┐
│ Payment Service   │
│ (High CPU Load)   │
└───────────────────┘

```
