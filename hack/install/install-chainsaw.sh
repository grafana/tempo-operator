#!/bin/bash
VERSION="0.1.3"

echo "Installing chainsaw"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="chainsaw"

url="https://github.com/kyverno/chainsaw/releases/download/v$VERSION/chainsaw_$(go env GOOS)_amd64"

download $PROGRAM $VERSION $url

export PATH=$PATH:/usr/local/bin

