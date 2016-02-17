#!/bin/bash
set -e

# RSI: sanitize project name, or leave that to go code?
project="$1"
volume="$2"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: rethinkdb-$project
  labels:
    k8s-app: rethinkdb
    project: $project
    version: v0
spec:
  replicas: 1
  selector:
    k8s-app: rethinkdb
    project: $project
    version: v0
  template:
    metadata:
      labels:
        k8s-app: rethinkdb
        project: $project
        version: v0
    spec:
      containers:
      - name: rethinkdb
        image: localhost:5000/rethinkdb
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
        awsElasticBlockStore:
          volumeID: $volume
          fsType: ext4

---

apiVersion: v1
kind: Service
metadata:
  name: rethinkdb-$project
  labels:
    k8s-app: rethinkdb
    project: $project
spec:
  selector:
    k8s-app: rethinkdb
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