#!/usr/bin/env bash
set -e

cd "./contract"

for yaml in *.yaml; do
  pkg="${yaml%.yaml}"
  outdir="./${pkg}"
  mkdir -p ${outdir}
  echo "Generating Go code for '$yaml' into package '$pkg'"

  ${GOPATH}/bin/oapi-codegen \
    --package="$pkg" \
    --config="$(pwd)/../.github/config/oapi-codegen.yml" \
    --generate "types,gin-server" \
    "$yaml" > "$outdir/$pkg.gen.go"
done