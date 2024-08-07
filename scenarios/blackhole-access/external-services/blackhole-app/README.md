
# Overview

This is a simple Node.js Express server that listens for incoming requests on a given port. 
For each request _path_ supported, it forwards the request back to the respective K8s Service. 
(e.g. `shipping.blackhole-external:8080/shipment` forwards to `shipping.blackhole-access:80/shipment` Service)

The intent is to simulate an application/service external from the Kubernetes cluster that needs to then 
communicate with services within the cluster - for example an Application Load Balancer service.

Note that the forward-to endpoint is via HTTP port 80. That is where the `/etc/hosts` file comes into play, once again resolving
the DNS lookup to your local host machine. Additionally, we need an associated `Ingress` object in our K8s cluster

See: [docs/blackhole-setup.md](../../docs/blackhole-setup.md) for a more complete picture of the overall setup.

## Setup

1. `npm install` - from within the `external-services/blackhole-app` directory
2. `npm start`

Everytime an incoming request is handled and forwarded, you will see a log message in the terminal with the path of that
request.