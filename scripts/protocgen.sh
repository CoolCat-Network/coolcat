#!/usr/bin/env bash

set -eo pipefail

echo "Generating gogo proto code"
cd proto
buf mod update
proto_dirs=$(find ./coolcat -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      buf generate --template buf.gen.gogo.yml $file
    fi
  done
done

cd ..

# move proto files to the right places
cp -r ./github.com/coolcat-network/coolcat/v1/x/* x/
rm -rf ./github.com