#!/bin/bash

if [ "$#" -ne 1 ]; then
  echo "Image Reference required"
  exit 1
fi

IMAGE_REF=$1
SKOPEO=$(command -v skopeo)
JQ=$(command -v jq)
SHA256=$(command -v sha256sum)

if [ "$SKOPEO" == "" ]; then
  echo "skopeo not found on path"
  exit 1
fi

if [ "$JQ" == "" ]; then
  echo "jq required"
  exit 1
fi

if [ "$SHA256" == "" ]; then
  echo "jq required"
  exit 1
fi

result=$($SKOPEO inspect --raw "docker://$IMAGE_REF")

if [ $? -ne 0 ]; then
  echo $result
  exit 1
fi

schemaVersion=$(echo $result | $JQ '.schemaVersion')

if [ "$schemaVersion" == "2" ]; then
  sha256=($(echo $result | sha256sum))
  echo -n $sha256
  exit 0
fi

result=$($SKOPEO --override-os linux inspect "docker://$IMAGE_REF")

if [ $? -ne 0 ]; then
  echo -n $result
  exit 1
fi

digest=$(echo $result | $JQ '.Digest')
echo -n $digest
exit 0
