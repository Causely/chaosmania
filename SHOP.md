
# Chaosmania Boutique Example

## Introduction
Chaosmania can be used to simulate fake applications with various problems running on Kubernetes. Specifically, we want to simulate problems that can impact the application performance and service availability.

## Scenarios
We are simulating a number of parallel shoppers adding items to carts inspired by the sample application published at https://github.com/GoogleCloudPlatform/microservices-demo and in the following examples we show two ways the checkout process can have problematic implementations:
1. the checkout process can access data from a relational database too frequently slowing down the processing
2. the checkout process can lock a datastructure in a downstream microservice for a long time slowing down the processing

In either cases, the shopping process can accumulate too much data, impacting the infrastructure and other services, while the checkout function cannot keep up.

## Deploy the boutique services to Kubernetes

```shell
helm upgrade --install --create-namespace --namespace chaosmania single ./helm/single --set otlp.endpoint=http://tempo.monitoring:4318
helm upgrade --install --create-namespace --namespace chaosmania boutique ./helm/boutique --set otlp.endpoint=http://tempo.monitoring:4318
```

## Start jobs to create either scenario

Start simulating shoppers adding items to the cart
```shell
helm upgrade --install --create-namespace --namespace chaosmania client-push ./helm/client  --set chaos.plan="/plans/shop.yaml"
```

Either simulate the checkout process with frequent data access
```shell
helm upgrade --install --create-namespace --namespace chaosmania client-pop ./helm/client  --set chaos.plan="/plans/checkout_sql.yaml"
```

Or simulate the checkout process with inefficient locking in a downstream service
```shell
helm upgrade --install --create-namespace --namespace chaosmania client-pop ./helm/client  --set chaos.plan="/plans/checkout_lock.yaml"
```
