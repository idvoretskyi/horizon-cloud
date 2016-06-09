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
  replicas: 1
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

      containers:
      - name: varnish
        image: `cat $gcr_id_path`
        resources:
          requests: { cpu: "250m" }
          limits: { memory: "768Mi" }
        readinessProbe:
          httpGet:
            port: 80
            path: /ebaefa90-3c6e-4eb4-b8d3-9e2d53aec696
        env:
        - name: VARNISH_MEMORY
          value: "512M"
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: names
          mountPath: /secrets/names
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
EOF
