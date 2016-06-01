#!/bin/bash
set -eu
set -o pipefail

DEPLOY=${DEPLOY-dev}

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

gcr_id_path=docker/$name/gcr_image_id

cat <<EOF
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: $name
  namespace: $DEPLOY
spec:
  replicas: 2
  template:
    metadata:
      labels:
        app: $name
    spec:
      volumes:
      - name: disable-api-access
        emptyDir: {}

      containers:
      - name: proxy
        image: `cat $gcr_id_path`
        resources:
          limits: { cpu: "250m", memory: "128Mi" }
        readinessProbe:
          tcpSocket:
            port: 8000
        env:
        - name: API_SERVER
          value: "http://hzc-api"
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        ports:
        - containerPort: 8000
          name: http
          protocol: TCP
EOF
