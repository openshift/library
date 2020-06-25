#!/usr/bin/env bash

echo "Creating data to check against"
make import DIR=_output

DOCUMENTS=($(echo "${1}" | tr ',' '\n'))
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