---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: productcatalog
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  rules:
    - host: productcatalog.blackhole-external
      http:
        paths:
          - pathType: Prefix
            path: "/prodcat"
            backend:
              service:
                name: productcatalog
                port:
                  number: 8080
