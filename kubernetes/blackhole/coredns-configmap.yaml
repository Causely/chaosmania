apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf {
           max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
        hosts custom.hosts productcatalog.chaos shipping.chaos {
          192.168.1.28 productcatalog.chaos
          192.168.1.28 shipping.chaos
          fallthrough
        }
    }
  hosts: |
    192.168.1.28 recommendation.chaos productcatalog.chaos shipping.chaos