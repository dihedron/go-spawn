#!/bin/bash

echo "Current environment:"
env
echo "Reading from STDIN:"
while read line; do
  echo "$line"
done < "${1:-/dev/stdin}"