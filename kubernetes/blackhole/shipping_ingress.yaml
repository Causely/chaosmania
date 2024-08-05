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
