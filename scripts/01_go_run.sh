#!/bin/bash
echo "## $0 received NUM ARGS : " $#
ENV_FILENAME='.env'
if [ $# -eq 1 ]; then
  ENV_FILENAME=${2:='.env'}
fi
echo "## will build go using env variables in ${ENV_FILENAME} ..."
if [ -r "$ENV_FILENAME" ]; then
  set -a
  source <(sed -e '/^#/d;/^\s*$/d' -e "s/'/'\\\''/g" -e "s/=\(.*\)/='\1'/g" $ENV_FILENAME )
  go run -v -ldflags="-X 'github.com/lao-tseu-is-alive/go-ai-llm-query/pkg/version.Build=$(date -Iseconds)'" cmd/toolCalling/toolCalling.go
  set +a
else
  echo "## 💥💥 env path argument : ${ENV_FILENAME} was not found"
  exit 1
fi


