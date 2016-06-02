#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"

cat <<EOF
      containers:
      - name: horizon
        image: $HORIZON_GCR_ID
        ${COMMAND-}
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        env:
        - name: HZ_SERVE_STATIC
          value: dist
        - name: HZ_DEBUG
          value: 'yes'
        - name: HZ_PERMISSIONS
          value: 'yes'
        - name: HZ_ALLOW_UNAUTHENTICATED
          value: 'yes'
        - name: HZ_ALLOW_ANONYMOUS
          value: 'yes'
        - name: HZ_SECURE
          value: 'no'
        - name: HZ_AUTO_CREATE_COLLECTION
          value: 'no'
        - name: HZ_AUTO_CREATE_INDEX
          value: 'yes'
        - name: HZ_CONNECT
          value: r-$project:28015
        - name: HZ_BIND
          value: 0.0.0.0
        ports:
        - containerPort: 8181
          name: horizon
          protocol: TCP
      volumes:
      - name: disable-api-access
        emptyDir: {}
EOF
