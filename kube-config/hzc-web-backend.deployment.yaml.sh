#!/bin/bash
set -eu
set -o pipefail

DEPLOY=${DEPLOY-dev}

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

gcr_id_path=docker/$name/gcr_image_id

replicas=1
if [ "$DEPLOY" == "prod" ]; then
    replicas=3
fi

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
  replicas: $replicas
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 50%
  template:
    metadata:
      labels:
        app: $name
    spec:
      volumes:
      - name: disable-api-access
        emptyDir: {}
      - name: names
        secret: { secretName: "names" }
      - name: api-keys
        secret: { secretName: "api-keys" }
      - name: wildcard-ssl
        secret: { secretName: "wildcard-ssl" }

      containers:
      - name: web-backend
        image: `cat $gcr_id_path`
        resources:
          requests: { cpu: "250m" }
          limits: { memory: "768Mi" }
# RSI: readinessProbe -- does httpGet work over https?
        env:
        - name: NODE_ENV
          value: "$node_env"
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: names
          mountPath: /secrets/names
        - name: api-keys
          mountPath: /secrets/api-keys
        - name: wildcard-ssl
          mountPath: /secrets/wildcard-ssl
        ports:
        - containerPort: 4433
          name: https
          protocol: TCP
EOF
