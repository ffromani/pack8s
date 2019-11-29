#!/bin/bash

set -ex

TESTDIR=$( mktemp -d /tmp/pack8s-functest-XXXXXXXX )

BASEPATH=$( dirname $( readlink -f $0 ) )

if [ ! -d kubevirtci ]; then
	git clone -q https://github.com/kubevirt/kubevirtci.git
fi

{
mkdir -p "$TESTDIR/bin"
cp pack8s "$TESTDIR/bin"
cp -a kubevirtci "$TESTDIR"
} > /dev/null

echo "export PACK8S_FUNCTEST_DIR=$TESTDIR"
