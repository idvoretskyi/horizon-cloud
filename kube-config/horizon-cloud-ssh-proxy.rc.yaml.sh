#!/bin/bash
set -eu
set -o pipefail

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

version=`cat $basename docker/$name/gcr_image_id | md5sum | head -c16`

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: $name--$version
  labels:
    app: $name
    version: "$version"
spec:
  replicas: 1
  selector:
    app: $name
    version: "$version"
  template:
    metadata:
      labels:
        app: $name
        version: "$version"
    spec:
      volumes:
      - name: ssh-proxy-keys
        secret: { secretName: "ssh-proxy-keys" }
      - name: api-shared-secret
        secret: { secretName: "api-shared-secret" }

      containers:
      - name: proxy
        image: `cat docker/$name/gcr_image_id`
        resources:
          limits: { cpu: "50m", memory: "128Mi" }
        env:
        - name: CLIENT_KEY
          value: /secrets/ssh-proxy-keys/client-rsa
        - name: HOST_KEY
          value: /secrets/ssh-proxy-keys/host-rsa
        - name: LISTEN
          value: ":22"
        - name: API_SERVER
          value: "http://horizon-api:8000"
        - name: API_SERVER_SECRET
          value: /secrets/api-shared-secret/api-shared-secret
        volumeMounts:
        - name: ssh-proxy-keys
          mountPath: /secrets/ssh-proxy-keys
        - name: api-shared-secret
          mountPath: /secrets/api-shared-secret
        ports:
        - containerPort: 22
          name: ssh
          protocol: TCP
EOF
