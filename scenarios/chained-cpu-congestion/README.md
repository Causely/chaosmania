# Payment Service CPU Overload Scenario

## Overview

This scenario is designed to simulate and analyze the impact of high CPU load in the Payment Service, which causes cascading delays and performance degradation in dependent services like the Order Service and Frontend Service. The plan is divided into two distinct phases:

	1.	Phase 1: Baseline Building
During this phase, the system is subjected to a low request rate with minimal latency (10 ms). This phase lasts for 8 minutes and is intended to establish baseline values for system performance under normal operating conditions. The Payment Service operates with low CPU usage, ensuring stable order processing and frontend service response times.
	2.	Phase 2: Increased Load and CPU Throttling
In this phase, the request rate is tripled, which leads to significant CPU throttling and high CPU utilization in the Payment Service. As a result, latency in the Order Service and Frontend Service increases dramatically, reaching up to 500 ms. This phase simulates a real-world scenario where a sudden spike in traffic causes the Payment Service to become a bottleneck, ultimately impacting the entire system’s performance.

The goal of this scenario is to observe how increased CPU load and memory leaks in the Payment Service affect the overall system, leading to delays in order processing and a degraded user experience on the frontend.

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
