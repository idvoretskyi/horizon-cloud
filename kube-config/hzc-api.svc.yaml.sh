#!/bin/bash
set -eu
set -o pipefail

cat <<EOF
apiVersion: v1
kind: Service
metadata:
  name: hzc-api
spec:
  type: ClusterIP
  ports:
  - port: 8000
    name: http
    protocol: TCP

---

apiVersion: v1
kind: Endpoints
metadata:
  name: hzc-api
subsets:
- addresses:
  - ip: api.`cat /secrets/names/domain`
  ports:
  - port: 8000
    name: http
EOF
