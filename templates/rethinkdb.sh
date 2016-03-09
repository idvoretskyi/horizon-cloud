#!/bin/bash
set -eu

# RSI: sanitize project name, or leave that to go code?
project="$1"
volume="$2"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: rethinkdb-1-$project
  labels:
    app: rethinkdb
    project: $project
    version: v1
spec:
  replicas: 1
  selector:
    app: rethinkdb
    project: $project
    version: v1
  template:
    metadata:
      labels:
        app: rethinkdb
        project: $project
        version: v1
    spec:
      containers:
      - name: rethinkdb
        image: us.gcr.io/horizon-cloud-1239/rethinkdb:1
        resources:
          limits:
            cpu: 250m
            memory: 512Mi
        volumeMounts:
        - name: data
          mountPath: /data
        ports:
        - containerPort: 28015
          name: driver
          protocol: TCP
        - containerPort: 29015
          name: intracluster
          protocol: TCP
        - containerPort: 8080
          name: webui
          protocol: TCP
      volumes:
      - name: data
        gcePersistentDisk:
          pdName: $volume
          fsType: ext4

---

apiVersion: v1
kind: Service
metadata:
  name: rethinkdb-$project
  labels:
    app: rethinkdb
    project: $project
spec:
  selector:
    app: rethinkdb
    project: $project
  ports:
  - port: 28015
    name: driver
    protocol: TCP
  - port: 29015
    name: intracluster
    protocol: TCP
  - port: 8080
    name: webui
    protocol: TCP
EOF
