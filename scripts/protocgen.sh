#!/usr/bin/env bash

set -e

GO_MOD_PACKAGE="github.com/webstack/webstack"

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    # Only generate gogo proto for files with go_package pointing to our module (not api/)
    if grep -q "option go_package" "$file" && grep -H -o -c "option go_package.*$GO_MOD_PACKAGE/api" "$file" | grep -q ':0$'; then
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

echo "Generating pulsar proto code"
buf generate --template buf.gen.pulsar.yaml

cd ..

# Move gogo generated files from full module path to repo root
cp -r $GO_MOD_PACKAGE/* ./
rm -rf github.com

# Copy pulsar files into api/ directory
rm -rf api && mkdir api
custom_modules=$(find . -name 'module' -type d -not -path "./proto/*" -not -path "./.cache/*")

# Get the base namespace (1 level up from module/, strip leading ./)
base_namespace=$(echo $custom_modules | sed -e 's|/module||g' | sed -e 's|\./||g')

for module in $base_namespace; do
  echo " [+] Moving: ./$module to ./api/$module"

  mkdir -p api/$module
  mv $module/* ./api/$module/

  # Fix incorrect SDK type references for pulsar generated files
  find api/$module -type f -name '*.go' -exec sed -i'' -e 's|types "github.com/cosmos/cosmos-sdk/types"|types "cosmossdk.io/api/cosmos/base/v1beta1"|g' {} \;
  find api/$module -type f -name '*.go' -exec sed -i'' -e 's|types1 "github.com/cosmos/cosmos-sdk/x/bank/types"|types1 "cosmossdk.io/api/cosmos/bank/v1beta1"|g' {} \;

  rm -rf $module
done
