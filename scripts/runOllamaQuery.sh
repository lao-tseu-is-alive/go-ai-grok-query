#!/bin/bash
# Check if a prompt was provided as an argument
if [ -z "$1" ]; then
  # Print error to stderr
  echo "## 💥💥 ERROR: No prompt provided." >&2
  echo "Usage: $0 \"Your question here\"" >&2
  exit 1
fi
PROMPT=$1
echo "{\"prompt\": \"${PROMPT}\"}"  >&2
curl -s http://localhost:11434/api/chat -d '{
  "model": "deepseek-r1:latest",
  "stream": false,
  "messages": [
   {
      "role": "system",
      "content": "You are a bash shell assistant."
   },
   {
      "role": "user",
      "content": "'"$PROMPT"'"
   }
  ]
}'
