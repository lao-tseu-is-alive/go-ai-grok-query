#!/bin/bash
echo "## $0 received NUM ARGS : " $#
ENV_FILENAME='.env'
if [ $# -eq 1 ]; then
  ENV_FILENAME=${2:='.env'}
fi
echo "## will run go test using env variables in ${ENV_FILENAME} ..."
if [ -r "$ENV_FILENAME" ]; then
  echo "## will execute $BIN_FILENAME"
  set -a
  source <(sed -e '/^#/d;/^\s*$/d' -e "s/'/'\\\''/g" -e "s/=\(.*\)/='\1'/g" $ENV_FILENAME )
  go test -coverprofile coverage.out ./...
  set +a
else
  echo "## 💥💥 env path argument : ${ENV_FILENAME} was not found"
  exit 1
fi


