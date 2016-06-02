#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: h0-$project
  labels:
    app: horizon
    project: $project
    version: v4
spec:
  replicas: 1
  selector:
    app: horizon
    project: $project
    version: v4
  template:
    metadata:
      labels:
        app: horizon
        project: $project
        version: v4
    spec:
`./horizon-spec.sh $project`

---

apiVersion: v1
kind: Service
metadata:
  name: h-$project
  labels:
    app: horizon
    project: $project
spec:
  type: ClusterIP
  selector:
    app: horizon
    project: $project
  ports:
  - port: 8181
    name: http
EOF
