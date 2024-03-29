#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"
volume="$2"

command='command: ["/bin/bash", "-c", "echo | hz set-schema -n app -"]'

cat <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: r0-$project
  labels:
    app: rethinkdb
    project: $project
    version: v1
spec:
  replicas: 1
  selector:
    app: rethinkdb
    project: $project
    version: v1
  template:
    metadata:
      labels:
        app: rethinkdb
        project: $project
        version: v1
    spec:
      containers:
      - name: rethinkdb
        image: $RETHINKDB_GCR_ID
        resources:
          limits:
            cpu: 250m
            memory: 512Mi
        volumeMounts:
        - name: disable-api-access
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        - name: data
          mountPath: /data
        env:
        - name: RDB_CACHE_SIZE
          value: "384"
        ports:
        - containerPort: 28015
          name: driver
          protocol: TCP
        - containerPort: 29015
          name: intracluster
          protocol: TCP
        - containerPort: 8080
          name: webui
          protocol: TCP
      volumes:
      - name: disable-api-access
        emptyDir: {}
      - name: data
        gcePersistentDisk:
          pdName: $volume
          fsType: ext4

---

apiVersion: v1
kind: Service
metadata:
  name: r-$project
  labels:
    app: rethinkdb
    project: $project
spec:
  selector:
    app: rethinkdb
    project: $project
  ports:
  - port: 28015
    name: driver
  - port: 8080
    name: webui

---

apiVersion: batch/v1
kind: Job
metadata:
  name: ss-$project
spec:
  activeDeadlineSeconds: 300
  template:
    metadata:
      name: ss-$project
    spec:
      restartPolicy: OnFailure
`COMMAND="$command" ./horizon-spec.sh "$project"`
EOF
