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
    type: Recreate
  template:
    metadata:
      labels:
        app: $name
    spec:
      volumes:
      - name: disable-api-access
        emptyDir: {}
      - name: api-shared-secret
        secret: { secretName: "api-shared-secret" }
      - name: token-secret
        secret: { secretName: "token-secret" }
      - name: names
        secret: { secretName: "names" }
      - name: gcloud-service-account
        secret: { secretName: "gcloud-service-account" }
      - name: ku-config
        secret: { secretName: "ku-config" }

      containers:
      - name: proxy
        image: `cat $gcr_id_path`
        resources:
          limits: { memory: "256Mi" }
        readinessProbe:
          tcpSocket:
            port: 8000
        env:
        - name: HZC_LISTEN
          value: ":8000"
        - name: HZC_SHARED_SECRET
          value: /secrets/api-shared-secret/api-shared-secret
        - name: HZC_TOKEN_SECRET
          value: /secrets/token-secret/token-secret
        - name: HZC_TEMPLATE_PATH
          value: /templates
        - name: HZC_STORAGE_BUCKET_FILE
          value: /secrets/names/storage-bucket
        - name: HZC_RETHINKDB_ADDR
          value: rethinkdb-sys:28015
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: api-shared-secret
          mountPath: /secrets/api-shared-secret
        - name: token-secret
          mountPath: /secrets/token-secret
        - name: names
          mountPath: /secrets/names
        - name: gcloud-service-account
          mountPath: /secrets/gcloud-service-account
        - name: ku-config
          mountPath: /home/hzc/.kube
        ports:
        - containerPort: 8000
          name: http
EOF
