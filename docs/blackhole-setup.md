
# Overview

This scenario demonstrates a business application which contains a chain of application to application accesses, but
where one of those accesses is to an external application/service outside of the Kubernetes cluster. The external service 
then sends traffic back in to the cluster to another service. The external service(s) and application(s) are collectively 
referred to as a "blackhole" in this scenario. The intent is to simulate a scenario such as an external 
Application Load Balancer (ALB) which may not be monitored/observed by the Kubernetes cluster.

## Expected Access Chain

The default scenario plan should produce the following topology access graph:

"recommendation" (app) -> accesses -> "productcatalog" (app) -> accesses -> "shipping" (app) -> accesses -> "ad" (app)

The key detail is that "productcatalog" actually accesses an external network endpoint "shipping.blackhole-external" 
which then sends traffic out of the cluster to an external NodeJS proxy application which then forwards the request back
into the cluster destined for the "shipping" service.

----

# Running the Scenario

1. Go to the `scenarios/blackhole-access` directory.
2. Review the `common_vars.sh` file and modify any variables as needed.
   1. Most of the defaults should be fine to use.
3. Run `./configure.sh` to interactively configure the scenario.
   1. Prompt to edit your `/etc/hosts` file
   2. Prompt to edit the `coredns` ConfigMap in the `kube-system` namespace.
   3. Prompt to add Ingress rules to your K8s cluster. (default you need to add "shipping_ingress.yaml" Ingress rule)
   4. The results of this will add files to the `scenarios/blackhole-access/conf` directory, used by the `run.sh` script
4. Run `./run.sh` to start the scenario.
   1. This will start the `blackhole-app` NodeJS service in your local environment.
   2. It will also start the `boutique` and `single` applications in your K8s cluster.
   3. Finally, it will start the `client` job with the plan to be executed.

# Stopping the Scenario

To cleanup all resources created by the scenario, run `./cleanup.sh`.

----

# Manual Configuration

The deployment example here assumes running the chaosmania services within the namespace `blackhole-access`.
All external services are assumed to use the domain `*.blachole-external` and all internal services are assumed to use the domain `*.blackhole-access`.

1. Open `scenarios/blackhole-access/plan.yaml` or `plans/blackhole_access.yaml` 
  - defines a chain of applications communicating to each other. The `url` values for each hop are either direct k8s services
    identified by their service name, or external endpoints identified by their DNS name.
  
    For example the default configuration has one external hop defined as:

    ```yaml
      url: http://shipping.blackhole-external:8080/shipment
    ```

2. The `coredns` ConfigMap in your `kube-system` namespace needs to be configured to include custom hosts entries for each external 
   endpoint DNS lookup. We need those DNS entries to resolve to the IP address of your local host machine on your network.
   
    Example:
    ```yaml
    hosts custom.hosts productcatalog.blackhole-external shipping.blackhole-external {
      192.168.1.28 productcatalog.blackhole-external
      192.168.1.28 shipping.blackhole-external
      fallthrough
    }
    ```

    There is a sample file at `kubernetes/blackhole/coredns-configmap.yaml`, however, it is probably best to copy and then edit
    your existing ConfigMap to ensure you don't lose any existing configuration. 
   - Make a copy of existing configmap: `kubectl -n kube-system get configmap coredns -o yaml > coredns-configmap.yaml`
   - Add the custom.hosts block with the necessary details, then `kubectl -n kube-system apply -f coredns-configmap.yaml`.

3. Your `/etc/hosts` file needs to resolve the external endpoints to your local host machine,
   and from external service back to the K8s Cluster. Example adds `shipping.blackhole-external` (from K8s) and `shipping.blackhole-access` (to K8s)
    ```shell
    ## Causely
    127.0.0.1        localhost causely.localhost productcatalog.blackhole-external shipping.blackhole-external productcatalog.blackhole-access shipping.blackhole-access
    255.255.255.255  broadcasthost
    ```

4. Add an `Ingress` object to your K8s cluster to route incoming traffic to the appropriate K8s service/application. 
   You will need one Ingress object defined per external-to-internal network endpoint. 

   For example:
    ```yaml
    ---
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: shipping
      annotations:
        nginx.ingress.kubernetes.io/ssl-redirect: "false"
    spec:
      rules:
        - host: shipping.blackhole-external
          http:
            paths:
              - pathType: Prefix
                path: "/shipment"
                backend:
                  service:
                    name: shipping
                    port:
                      number: 8080
    ```
    This Ingress rule routes traffic from `shipping.blackhole-access:80/shipment` to the `shipping` Service on port 8080.

5. Run the "blackhole service/application" in your local environment. Open a terminal to `external-services/blackhole-app`
   and run it:
    ```shell
    npm install
    npm start
    ```
   see: [external-services/blackhole-app/README.md](../../external-services/blackhole-app/README.md) for more details.


6. Modify the `helm/boutique/values.yaml` and `helm/single/values.yaml` files to enable OpenTelemetry traces to be sent 
to your local opentelementry collector.
    ```yaml
    otlp:
      enabled: true
      endpoint: "http://opentelemetry-collector.default:4318"
      insecure: true
    ```

7. Run the `boutique` and `single` applications in your K8s cluster, in the `"blackhole-access"` namespace. See the `helm` directory for more details.
8. Run the `client` job for the blackhole scenario/plan.
```shell
helm upgrade --install --create-namespace --namespace blackhole-access client ./helm/client  --set chaos.plan="plans/blackhole_access.yaml"
```

The plan default runs one client instance sending requests to `recommendation` - to - `productcatalog` - to - `shipping.chaos` (external) - to - `shipping` services,
every 10 seconds. 