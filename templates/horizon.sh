#!/bin/bash
set -eu

# RSI: sanitize project name, or leave that to go code?
project="$1"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: horizon-4-$project
  labels:
    app: horizon
    project: $project
    version: v4
spec:
  replicas: 1
  selector:
    app: horizon
    project: $project
    version: v4
  template:
    metadata:
      labels:
        app: horizon
        project: $project
        version: v4
    spec:
      containers:
      - name: horizon
        image: us.gcr.io/horizon-cloud-1239/horizon:3
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
        env:
        - name: HZ_SERVE_STATIC
          value: dist
        - name: HZ_DEBUG
          value: 'true'
        - name: HZ_ALLOW_UNAUTHENTICATED
          value: 'true'
        - name: HZ_INSECURE
          value: 'true'
        - name: HZ_AUTO_CREATE_TABLE
          value: 'true'
        - name: HZ_AUTO_CREATE_INDEX
          value: 'true'
        - name: HZ_CONNECT
          value: rethinkdb-$project:28015
        - name: HZ_BIND
          value: 0.0.0.0
        ports:
        - containerPort: 8181
          name: horizon
          protocol: TCP

---

apiVersion: v1
kind: Service
metadata:
  name: horizon-$project
  labels:
    app: horizon
    project: $project
spec:
  selector:
    app: horizon
    project: $project
  ports:
  - port: 8181
    name: http
    protocol: TCP
  type: ClusterIP
EOF
