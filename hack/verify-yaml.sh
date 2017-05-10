#!/usr/bin/env bash

echo "Checking official.yaml for syntax errors."
ret=0
python -c 'import yaml,sys;yaml.safe_load(sys.stdin)' < official.yaml || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "SUCCESS: No YAML syntax errors detected."
else
  echo "FAILURE: YAML syntax v/errors detected!"
  exit 1
fi

echo "Checking community.yaml for syntax errors."
ret=0
python -c 'import yaml,sys;yaml.safe_load(sys.stdin)' < community.yaml || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "SUCCESS: No YAML syntax errors detected."
else
  echo "FAILURE: YAML syntax errors detected!"
  exit 1
fi
