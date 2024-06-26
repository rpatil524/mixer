#!/bin/bash
#
# A helper script to update the storage versions and prepare a commit for
# release.
#

DIR=$(dirname "$0")

cd "$DIR"/..

function update_version() {
  echo ""
  echo "==== Updating BT and BQ versions ===="

  yq eval -i 'del(.tables)' deploy/storage/base_bigtable_info.yaml
  yq eval -i '.tables = []' deploy/storage/base_bigtable_info.yaml
  for src in $(gsutil ls gs://datcom-control/autopush/*_latest_base_cache_version.txt | grep -v experimental | sort); do
    echo "Copying $src"
    export TABLE="$(gsutil cat "$src")"
     yq eval -i '.tables += [env(TABLE)]' deploy/storage/base_bigtable_info.yaml
  done

  BQ=$(gsutil cat gs://datcom-control/latest_base_bigquery_version.txt)
  printf "$BQ" > deploy/storage/bigquery.version
}

function update_proto() {
  echo ""
  echo "==== Updating go proto files ===="
  protoc \
    --proto_path=proto \
    --go_out=paths=source_relative:internal/proto \
    --go-grpc_out=paths=source_relative:internal/proto \
    --go-grpc_opt=require_unimplemented_servers=false \
    --experimental_allow_proto3_optional \
    --include_imports \
    --include_source_info \
    --descriptor_set_out mixer-grpc.pb \
    proto/*.proto proto/**/*.proto

  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to update proto"
    exit 1
  fi
}

function update_golden() {
  echo ""
  echo "==== Updating staging golden files ===="
  ./scripts/update_golden.sh
  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to update proto"
    exit 1
  fi
}

function commit() {
  echo ""
  echo "==== Committing the change ===="
  git commit -a -m "Data Release: $(date +%F)"
}

update_version
# update_proto
update_golden
commit

echo ""
echo "NOTE: Please review the commit, push to your remote repo and create a PR."
echo ""
