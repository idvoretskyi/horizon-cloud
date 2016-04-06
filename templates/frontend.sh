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
  name: f0-$project
  namespace: user
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
        image: `cat ../kube-config/docker/horizon-nginx/gcr_image_id_$cluster_name`
        resources:
          limits:
            cpu: 50m
            memory: 128Mi # must be set high to avoid attached volume limits
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
        image: `cat ../kube-config/docker/horizon-openssh/gcr_image_id_$cluster_name`
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
  namespace: user
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
  namespace: user
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
