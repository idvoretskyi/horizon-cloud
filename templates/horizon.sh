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
        image: $HORIZON_GCR_ID
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        env:
        - name: HZ_SERVE_STATIC
          value: dist
        - name: HZ_DEBUG
          value: 'yes'
        - name: HZ_PERMISSIONS
          value: 'no'
        - name: HZ_ALLOW_UNAUTHENTICATED
          value: 'yes'
        - name: HZ_ALLOW_ANONYMOUS
          value: 'yes'
        - name: HZ_SECURE
          value: 'no'
        - name: HZ_AUTO_CREATE_COLLECTION
          value: 'yes'
        - name: HZ_AUTO_CREATE_INDEX
          value: 'yes'
        - name: HZ_CONNECT
          value: r-$project:28015
        - name: HZ_BIND
          value: 0.0.0.0
        ports:
        - containerPort: 8181
          name: horizon
          protocol: TCP
      volumes:
      - name: disable-api-access
        emptyDir: {}

---

apiVersion: v1
kind: Service
metadata:
  name: h-$project
  labels:
    app: horizon
    project: $project
spec:
  type: ClusterIP
  selector:
    app: horizon
    project: $project
  ports:
  - port: 8181
    name: http
EOF
