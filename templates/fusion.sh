#!/bin/bash
set -e

# RSI: sanitize project name, or leave that to go code?
project="$1"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: fusion-$project
  labels:
    app: fusion
    project: $project
    version: v2
spec:
  replicas: 1
  selector:
    app: fusion
    project: $project
    version: v2
  template:
    metadata:
      labels:
        app: fusion
        project: $project
        version: v2
    spec:
      containers:
      - name: fusion
        image: localhost:5000/fusion:2
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
        env:
        - name: FUSION_CONNECT
          value: rethinkdb-$project:28015
        ports:
        - containerPort: 8181
          name: fusion
          protocol: TCP

---

apiVersion: v1
kind: Service
metadata:
  name: fusion-$project
  labels:
    app: fusion
    project: $project
spec:
  selector:
    app: fusion
    project: $project
  ports:
  - port: 8181
    name: driver
    protocol: TCP
  type: LoadBalancer
EOF
