#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: h0-$project
  namespace: user
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
        image: `cat ../kube-config/docker/horizon/gcr_image_id`
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
          value: r-$project:28015
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
  name: h-$project
  namespace: user
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
