#!/bin/bash
set -ue
set -x
dir=$(pwd)
cd $dir/proto/listify
buf push
cd $dir/proto/protostate
buf push
cd $dir
buf generate ./proto/listify
buf generate ./proto/protostate
cd $dir/internal/testproto
buf mod update
buf generate
cd $dir
