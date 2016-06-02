#!/bin/bash
set -eu
set -o pipefail

cd "$(dirname "$(readlink -f "$0")")"

project="$1"
command='command: ["/bin/bash", "-c", "echo | hz set-schema -n app -"]'

cat <<EOF
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
