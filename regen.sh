#!/bin/bash
set -ue
set -x
dir=$(pwd)

PULL=false
PUSH=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --pull)
      PULL=true
      shift # past argument
      ;;
    --push)
      PUSH=true
      shift # past argument
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      POSITIONAL_ARGS+=("$1") # save positional arg
      shift # past argument
      ;;
  esac
done


cd $dir/proto/listify
if [ $PULL = true ]; then
  buf mod update
fi
if [ $PUSH = true ]; then
  buf push
fi
cd $dir/proto/protostate
if [ $PULL = true ]; then
  buf mod update
fi
if [ $PUSH = true ]; then
  buf push
fi
cd $dir

buf generate ./proto/listify
buf generate ./proto/protostate

cd $dir/testproto
if [ $PULL = true ]; then
  buf mod update
fi
buf generate
cd $dir
