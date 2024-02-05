
# Setup local (strimzi) Kafka cluster

## Prerequisites

### Edit [helm/video/values.yaml](../helm/video/values.yaml)

#### Set `datadog.enabled` to `false`

#### Configure Consumer Clients

There are (as of writing) 3 `background_services`

    - name: "encoder-data-broker-consumer"
    - name: "upload-message-broker-consumer"
    - name: "inventory-message-broker-consumer"

for each of these they need to be configured to use the Kafka cluster, e.g.
```yaml
  - name: "encoder-data-broker-consumer"
    type: kafka-consumer
    config:
      tracing_service_name: data-broker
      brokers:
        - "chaosmania-kafka-cluster-kafka-brokers:9092"
      username: "" # TODO
      password: "" # TODO
      tls_enable: false
      sasl_enable: false
      topic: test1
      group: encoder-group
```
**Notes:** 
  - `type` should be `kafka-consumer` here. (options are: `rabbitmq-consumer` | `kafka-producer`)
  - `username` and `password` are not used in the current setup.
  - `tls_enable` and `sasl_enable` are not used in the current setup.
  - `topic` and `group` will defer depending on the producer - consumer relationship/script.
  - `tracing_service_name` may not be important as we should be using the `peer.service` derived from the broker(s).

#### Configure Producer Clients

Similar to `background_services` config, under the `services:` block edit the:
```yaml
  - name: "data-broker-producer"
    type: kafka-producer
    config:
      tracing_service_name: data-broker
      brokers:
        - "chaosmania-kafka-cluster-kafka-brokers:9092"
      username: "" # TODO
      password: "" # TODO
      tls_enable: false
      sasl_enable: false
      topic: test1
```

#### Configure OTLP (optional)

example:
```yaml
otlp:
  enabled: true
  endpoint: "http://otel-collector.otel-collector.svc.cluster.local:4317"
```

---

## Install Strimzi Kafka Operator

There are some convenient scripts in the `scripts` directory to help with managing this.  
- [create_kafka_cluster.sh](../scripts/create_kafka_cluster.sh)
- [delete_kafka_cluster.sh](../scripts/delete_kafka_cluster.sh)

#### Edit the `create_kafka_cluster.sh` script

- set `NS` var to the namespace you want to install the Kafka cluster into. (default: `chaosmania`)
- execute `./scripts/create_kafka_cluster.sh` to install the Kafka cluster.

You _should_ end up with the following resources:

`kubectl -n chaosmania get pods`
```kubectl
NAME                                                          READY  
| chaosmania-kafka-cluster-chaosmania-kafka-pool-0          ●   1/1    
│ chaosmania-kafka-cluster-chaosmania-kafka-pool-1          ●   1/1    
│ chaosmania-kafka-cluster-chaosmania-kafka-pool-2          ●   1/1    
│ chaosmania-kafka-cluster-entity-operator-b89d77fd9-szt22  ●   1/1 
| strimzi-cluster-operator-868f55785f-z5hqj                 ●   1/1    
```

`kubectl -n chaosmania get svc`
```kubectl
chaosmania-kafka-cluster-kafka-bootstrap   ClusterIP   10.96.45.138    <none>        9091/TCP,9092/TCP,9093/TCP                     85m
chaosmania-kafka-cluster-kafka-brokers     ClusterIP   None            <none>        9090/TCP,9091/TCP,8443/TCP,9092/TCP,9093/TCP   85m
```
**Notes:** 
- The `...-kafka-brokers` service is what we use for the `brokers` lists config in the `values.yaml` file.

`kubectl -n chaosmania get kafkatopics`
```kubectl
NAME                       CLUSTER                    PARTITIONS   REPLICATION FACTOR   READY
inventory-video-encoding   chaosmania-kafka-cluster   1            3                    True
test1                      chaosmania-kafka-cluster   1            3                    True
upload-video-encoding      chaosmania-kafka-cluster   1            3                    True
```

---

## Run the `chaosmania` helm `video` deployment

Something like this (note the `--set image.tag=xxxx` or just use `latest`):
- `helm upgrade --install --create-namespace --namespace chaosmania video ./helm/video --set image.tag=brandon`

---

## Run the plan/plans you want to load

- open [update_video_clients.sh](../update_video_clients.sh) and set the variables appropriately.
- execute `./update_video_clients.sh` to run the plan(s) via helm deployments
