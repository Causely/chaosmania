---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: recommendation
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  rules:
    - host: recommendation.blackhole-external
      http:
        paths:
          - pathType: Prefix
            path: "/recommends"
            backend:
              service:
                name: recommendation
                port:
                  number: 8080
