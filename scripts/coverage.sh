#!/usr/bin/env sh
set -eu

PACKAGE_NAME=$(basename "$PWD")

COVERAGE_OUT=coverage.tmp.out
COVERAGE_RESULT=coverage.out

if [ -f "$COVERAGE_RESULT" ]; then
  rm -f $COVERAGE_RESULT
fi

echo "mode: atomic" > $COVERAGE_RESULT
for pkg in $(GOPATH="$PWD:$PWD/vendor" go list "$PACKAGE_NAME"/...); do
  GOPATH="$PWD:$PWD/vendor" go test -v -race -cover -covermode=atomic -coverprofile=$COVERAGE_OUT $pkg
  if [ -f $COVERAGE_OUT ]; then
    sed -i -e "s/^/github.com\/zchee\/$PACKAGE_NAME\/src\//g" $COVERAGE_OUT
    tail -n +2 $COVERAGE_OUT >> $COVERAGE_RESULT
    rm -f $COVERAGE_OUT
  fi
done
