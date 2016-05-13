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
  replicas: 2
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
      - name: disable-api-access
        emptyDir: {}
      - name: names
        secret: { secretName: "names" }

      containers:
      - name: varnish
        image: `cat $gcr_id_path`
        resources:
          requests: { cpu: "250m" }
          limits: { memory: "256Mi" }
        env:
        - name: VARNISH_MEMORY
          value: "128M"
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
