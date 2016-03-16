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
      - name: api-shared-secret
        secret: { secretName: "api-shared-secret" }
      - name: hzcio-ssl
        secret: { secretName: "hzcio-ssl" }

      containers:
      - name: proxy
        image: `cat docker/$name/gcr_image_id`
        resources:
          limits: { cpu: "250m", memory: "128Mi" }
        env:
        - name: API_SERVER
          value: "http://horizon-api:8000"
        - name: SECRET_PATH
          value: /secrets/api-shared-secret/api-shared-secret
        volumeMounts:
        - name: api-shared-secret
          mountPath: /secrets/api-shared-secret
        - name: hzcio-ssl
          mountPath: /secrets/hzcio-ssl
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
        - containerPort: 443
          name: https
          protocol: TCP
EOF
