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
      - name: ssh-proxy-keys
        secret: { secretName: "ssh-proxy-keys" }
      - name: api-shared-secret
        secret: { secretName: "api-shared-secret" }
      - name: token-secret
        secret: { secretName: "token-secret" }

      containers:
      - name: proxy
        image: `cat $gcr_id_path`
        resources:
          limits: { cpu: "50m", memory: "128Mi" }
        readinessProbe:
          tcpSocket:
            port: 2222
        env:
        - name: HOST_KEY
          value: /secrets/ssh-proxy-keys/host-rsa
        - name: LISTEN
          value: ":2222"
        - name: API_SERVER
          value: "http://hzc-api"
        - name: API_SERVER_SECRET
          value: /secrets/api-shared-secret/api-shared-secret
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: ssh-proxy-keys
          mountPath: /secrets/ssh-proxy-keys
        - name: api-shared-secret
          mountPath: /secrets/api-shared-secret
        - name: token-secret
          mountPath: /secrets/token-secret
        ports:
        - containerPort: 2222
          name: ssh
EOF
