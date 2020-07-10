#!/usr/bin/env bash

start=`date +%s`

echo "Creating data to check against"
./library import --config configs/verify-cluster-samples-operator-periodic.yaml

DOCUMENTS=($(echo "official,community,arch,operator,online" | tr ',' '\n'))
for document in "${DOCUMENTS[@]}"
do
  echo "Comparing current ${document} directory and freshly generated ${document} directory"
  ret=0
  diff -Naupr "${document}" "_output/${document}" || ret=$?
  if [[ $ret -eq 0 ]]
  then
    echo "SUCCESS: ${document} directory up to date."
  else
    echo "FAILURE: ${document} directory out of date. Please run make import"
    exit 1
  fi
done

echo "Cleaning up"
rm -rf _output

echo "Verification ran in $((`date +%s`-start)) seconds"