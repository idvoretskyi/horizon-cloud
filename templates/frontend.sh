#!/bin/bash
set -eu

# RSI: sanitize project name, or leave that to go code?
project="$1"
volume="$2"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: f0-$project
  labels:
    app: frontend
    project: $project
    version: v1
spec:
  replicas: 1
  selector:
    app: frontend
    project: $project
    version: v1
  template:
    metadata:
      labels:
        app: frontend
        project: $project
        version: v1
    spec:
      volumes:
      - name: data
        gcePersistentDisk:
          pdName: $volume
          fsType: ext4

      containers:
      - name: nginx
        image: us.gcr.io/horizon-cloud-1239/horizon-nginx:1
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
        volumeMounts:
        - name: data
          readOnly: true
          mountPath: /data
        env:
        - name: NGINX_CONNECT
          value: h-$project:8181
        ports:
        - containerPort: 80
          name: http
          protocol: TCP

      - name: ssh
        image: us.gcr.io/horizon-cloud-1239/horizon-ssh:1
        resources:
          limits:
            cpu: 10m
            memory: 64Mi
        volumeMounts:
        - name: data
          mountPath: /data

---

apiVersion: v1
kind: Service
metadata:
  name: fn-$project
  labels:
    app: frontend
    project: $project
spec:
  selector:
    app: frontend
    project: $project
  ports:
  - port: 80
    name: http
    protocol: TCP
  type: ClusterIP

---

apiVersion: v1
kind: Service
metadata:
  name: fs-$project
  labels:
    app: frontend
    project: $project
spec:
  selector:
    app: frontend
    project: $project
  ports:
  - port: 22
    name: ssh
    protocol: TCP
  type: ClusterIP
EOF
