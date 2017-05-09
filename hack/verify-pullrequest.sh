#!/usr/bin/env bash

echo "Creating _output directory"
mkdir _output

echo "Copying files to _output directory"
rsync -av . _output --exclude _output

echo "Generating fresh content"
cd _output && ./import_content.py
cd ..

TMP_GENERATED_OFFICIAL_DIR="_output/official"
TMP_GENERATED_COMMUNITY_DIR="_output/community"

echo "Diffing current official directory against freshly generated official directory"
ret=0
diff -Naupr "official" "${TMP_GENERATED_OFFICIAL_DIR}" || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "SUCCESS: Generated official directory up to date."
else
  echo "FAILURE: Generated official directory out of date. Please run import_content.py"
  exit 1
fi

echo "Diffing current community directory against freshly generated community directory"
ret=0
diff -Naupr "community" "${TMP_GENERATED_COMMUNITY_DIR}" || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "SUCCESS: Generated community directory up to date."
else
  echo "FAILURE: Generated community directory out of date. Please run import_content.py"
  exit 1
fi
