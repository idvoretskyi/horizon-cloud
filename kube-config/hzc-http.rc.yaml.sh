#!/bin/bash
set -eu
set -o pipefail

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

gcr_id_path=docker/$name/gcr_image_id_`cat /secrets/names/cluster`
version=`cat $basename $gcr_id_path | md5sum | head -c16`

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
      - name: wildcard-ssl
        secret: { secretName: "wildcard-ssl" }

      containers:
      - name: proxy
        image: `cat $gcr_id_path`
        resources:
          limits: { cpu: "250m", memory: "128Mi" }
        env:
        - name: API_SERVER
          value: "http://api.`cat /secrets/names/domain`:8000"
        - name: SECRET_PATH
          value: /secrets/api-shared-secret/api-shared-secret
        volumeMounts:
        - name: api-shared-secret
          mountPath: /secrets/api-shared-secret
        - name: wildcard-ssl
          mountPath: /secrets/wildcard-ssl
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        - containerPort: 4433
          name: https
          protocol: TCP
EOF
