#!/bin/bash
set -eu
set -o pipefail

DEPLOY=${DEPLOY-dev}

basename=`basename $0`
name=${basename%%.*}

cd "$(dirname "$(readlink -f "$0")")"

gcr_id_path=docker/$name/gcr_image_id

hzc_http_host=hzc-http.$(cat /secrets/"$DEPLOY"/names/domain)

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
      - name: wildcard-ssl
        secret: { secretName: "wildcard-ssl" }

      containers:
      - name: stunnel
        image: `cat $gcr_id_path`
        resources:
          requests: { cpu: "250m" }
          limits: { memory: "256Mi" }
        readinessProbe:
          tcpSocket:
            port: 443
        env:
        - name: TARGET
          value: "$hzc_http_host:80"
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: wildcard-ssl
          mountPath: /secrets/wildcard-ssl
        ports:
        - containerPort: 443
          name: https
EOF
