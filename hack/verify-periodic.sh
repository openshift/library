#!/usr/bin/env bash

start=`date +%s`

echo "Creating data to check against"
./library import --documents official,community --tags arch_x86_64 --dir _output/arch/x86_64
./library import --documents official,community --tags arch_ppc64le --dir _output/arch/ppc64le
./library import --documents official,community --tags arch_s390x --dir _output/arch/s390x
./library import --documents official,community --tags okd,arch_x86_64 --dir _output/operator/okd-x86_64 --match-all-tags
./library import --documents official,community --tags ocp,arch_x86_64 --dir _output/operator/ocp-x86_64 --match-all-tags
./library import --documents official,community --tags ocp,arch_ppc64le --dir _output/operator/ocp-ppc64le --match-all-tags
./library import --documents official,community --tags ocp,arch_s390x --dir _output/operator/ocp-s390x --match-all-tags
./library import --documents official,community --tags online-starter,arch_x86_64 --dir _output/online/starter/x86_64 --match-all-tags
./library import --documents official,community --tags online-professional,arch_x86_64 --dir _output/online/professional/x86_64 --match-all-tags
./library import --documents official,community --dir _output

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