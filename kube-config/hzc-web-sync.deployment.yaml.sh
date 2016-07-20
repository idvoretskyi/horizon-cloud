#!/bin/bash
set -eu
set -o pipefail

DEPLOY=${DEPLOY-dev}

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

gcr_id_path=docker/$name/gcr_image_id

node_env=development
if [ "$DEPLOY" == "prod" ]; then
    node_env=production
fi

cat <<EOF
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: $name
  namespace: $DEPLOY
spec:
# There should NEVER be more than one of these, or they might
# clobber each other's inserts during generation cleanup and end in an
# invalid state.
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 0
  template:
    metadata:
      labels:
        app: $name
    spec:
      volumes:
      - name: disable-api-access
        emptyDir: {}
      - name: api-keys
        secret: { secretName: "api-keys" }

      containers:
      - name: proxy
        image: `cat $gcr_id_path`
        resources:
          limits: { cpu: "250m", memory: "128Mi" }
        env:
        - name: NODE_ENV
          value: "$node_env"
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: api-keys
          mountPath: /secrets/api-keys
EOF
