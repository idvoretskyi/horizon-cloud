#!/bin/bash
set -eu

DEPLOY=${DEPLOY-dev}

cat <<EOF
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: rethinkdb-sys
  namespace: $DEPLOY
spec:
  replicas: 1
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: rethinkdb-sys
    spec:
      volumes:
      - name: rethinkdb-sys-$DEPLOY
        gcePersistentDisk:
          pdName: rethinkdb-sys-$DEPLOY
          fsType: ext4
      containers:
      - name: rethinkdb
        image: rethinkdb:2.3.2
        resources:
          limits: { memory: 2000Mi }
          requests: { memory: 2000Mi, cpu: 500m }
        readinessProbe:
          tcpSocket:
            port: 28015
        ports:
        - containerPort: 28015
          name: driver
        - containerPort: 29015
          name: intracluster
        - containerPort: 8080
          name: webui
        volumeMounts:
        - name: rethinkdb-sys-$DEPLOY
          mountPath: /data
        command: ["rethinkdb", "--bind", "all", "--cache-size", "2000"]
EOF
