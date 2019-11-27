#!/bin/bash

set -e

BASEPATH=$( dirname $( readlink -f $0 ) )

if [ -z "$1" ]; then
	echo "usage: $0 provider1 [provider2 [... providerN]]" 1>&2
	exit 1
fi

eval $( $BASEPATH/mk-test-dir.sh )

if [ -z "${PACK8S_FUNCTEST_DIR}" ]; then
	exit 200
fi

for provider in $@; do
	$BASEPATH/runner.py $provider || exit 100
done

rm -rf ${PACK8S_FUNCTEST_DIR}
