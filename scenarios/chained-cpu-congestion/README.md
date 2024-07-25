# Payment Service CPU Overload Scenario

## Overview

This scenario simulates high CPU load in the Payment Service, which causes cascading delays and performance issues in dependent services such as the Order Service and Frontend Service. The goal is to identify how high CPU usage in the Payment Service impacts the overall system, including order processing delays and degraded user experience on the frontend.

## Topology

- **Inventory Service**: Manages product inventory.
- **Order Service**: Manages order processing and depends on the Inventory Service.
- **Payment Service**: Handles payment transactions and depends on the Order Service.
- **Frontend Service**: Provides the user-facing interface and depends on the Order Service and Payment Service.

## Objective

To test the resilience and performance of the microservices architecture by inducing high CPU load in the Payment Service and observing the resulting delays and performance degradation in the Order Service and Frontend Service.

## Folder Structure

```plaintext
scenarios/
├── payment-service-cpu-overload/
│   ├── plan.yaml
│   ├── run.sh
│   ├── README.md
```

## Configuration Files

- **plan.yaml**

This file defines the ChaosMania plan for inducing high CPU load in the Payment Service. The plan includes actions to simulate inefficient algorithms and memory leaks that cause the CPU load to increase progressively over time.

- **prun.sh**

This script sets up the environment, deploys the necessary services, and runs the ChaosMania scenario defined in the plan.yaml file.

## Error Propagation Schema
```plaintext
┌───────────────────┐
│                   │
│ Payment Service   │
│ (High CPU Load)   │
└───────────────────┘
         │
         ▼
┌───────────────────┐
│                   │
│ Order Service     │
│ (Delayed Orders)  │
└───────────────────┘
         │
         ▼
┌───────────────────┐
│                   │
│ Frontend Service  │
│ (Slow Responses)  │
└───────────────────┘
```

## Monitoring Metrics

Order Service

	•	High Latency: Increased response time for order processing requests.
	•	Time Out Errors: Higher rate of timeout errors in order processing requests.
	•	Increased Queue Length: Number of pending orders or length of order processing queue.

Frontend Service

	•	Slow Checkout Process: Increased response time for checkout-related API calls.
	•	Error Messages: Higher rate of error responses during the checkout process.
	•	Unresponsive UI: Increased frontend page load times and interaction response times.