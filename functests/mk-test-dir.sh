#!/bin/bash

set -ex

TESTDIR=$( mktemp -d /tmp/pack8s-functest-XXXXXXXX )

{
mkdir -p "$TESTDIR/bin"
cp pack8s "$TESTDIR/bin"

pushd .
cd "$TESTDIR"
git clone -q https://github.com/kubevirt/kubevirtci.git
popd
} > /dev/null

echo "export PACK8S_FUNCTEST_DIR=$TESTDIR"
