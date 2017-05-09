#!/usr/bin/env bash

echo "Checking for Python errors using pylint"
ret=0
pylint -E import_content.py || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "SUCCESS: No Python errors detected."
else
  echo "FAILURE: Python errors detected!"
  exit 1
fi