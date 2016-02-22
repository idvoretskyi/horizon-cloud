#!/bin/bash
set -e

# RSI: sanitize project name, or leave that to go code?
project="$1"
volume="$2"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: frontend-$project
  labels:
    k8s-app: frontend
    project: $project
    version: v0
spec:
  replicas: 1
  selector:
    k8s-app: frontend
    project: $project
    version: v0
  template:
    metadata:
      labels:
        k8s-app: frontend
        project: $project
        version: v0
    spec:
      volumes:
      - name: data
        awsElasticBlockStore:
          volumeID: $volume
          fsType: ext4
      - name: sshhostkeys
        secret:
          secretName: fusion-$project-sshhost

      containers:
      - name: nginx
        image: localhost:5000/fusion-nginx
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
          value: fusion-$project:8181
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
        - containerPort: 443
          name: https
          protocol: TCP

      - name: ssh
        image: localhost:5000/fusion-ssh
        resources:
          limits:
            cpu: 10m
            memory: 64Mi
        volumeMounts:
        - name: data
          mountPath: /data
        - name: sshhostkeys
          mountPath: /secrets

---

apiVersion: v1
kind: Service
metadata:
  name: frontend-nginx-$project
  labels:
    k8s-app: frontend
    project: $project
spec:
  selector:
    k8s-app: frontend
    project: $project
  ports:
  - port: 80
    name: http
    protocol: TCP
  - port: 443
    name: https
    protocol: TCP
  type: LoadBalancer

---

apiVersion: v1
kind: Service
metadata:
  name: frontend-ssh-$project
  labels:
    k8s-app: frontend
    project: $project
spec:
  selector:
    k8s-app: frontend
    project: $project
  ports:
  - port: 22
    name: ssh
    protocol: TCP
  type: NodePort
EOF
