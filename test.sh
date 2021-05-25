#!/bin/bash
# test.sh $METHOD $PATH $HEADER_KEY_1 $HEADER_VALUE_1 $HEADER_KEY_2 $HEADER_VALUE_2 ...

echo "Method: $1"
echo "Path: $2"
echo "Headers: "

shift 2
while test ${#} -gt 0
do
  echo "  $1: $2"
  shift 2
done

echo -n "Body: "
while IFS= read -r LINE || [[ -n "$LINE" ]]; do
    echo -n "$LINE"
done
echo ""