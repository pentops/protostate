#!/bin/bash
dir=$(pwd)
cd $dir/proto
buf push
cd $dir
buf generate ./proto
cd $dir/testproto
buf mod update
buf generate
cd $dir
