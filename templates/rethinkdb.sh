#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"
volume="$2"
cluster_name=`cat /secrets/names/cluster`

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: r0-$project
  namespace: user
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
        image: `cat ../kube-config/docker/rethinkdb/gcr_image_id_$cluster_name`
        resources:
          limits:
            cpu: 250m
            memory: 512Mi
        volumeMounts:
        - name: data
          mountPath: /data
        env:
        - name: RDB_CACHE_SIZE
          value: "384"
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
  name: r-$project
  namespace: user
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
  - port: 8080
    name: webui
    protocol: TCP
EOF
