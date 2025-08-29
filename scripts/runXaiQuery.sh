#!/bin/bash
# Check if a prompt was provided as an argument
if [ -z "$1" ]; then
  # Print error to stderr
  echo "## 💥💥 ERROR: No prompt provided." >&2
  echo "Usage: $0 \"Your question here\"" >&2
  exit 1
fi
# Check if XAI_API_KEY is already set in the environment
if [ -z "${XAI_API_KEY}" ]; then
  # If not set, check for a .env file in the current directory
  if [ -f ".env" ]; then
    echo "INFO: XAI_API_KEY not set, sourcing from .env file."
    # Source the .env file to load the variables into the current shell
    source .env
  else
    # If no .env file is found, exit with an error
    echo "## 💥💥 ERROR: XAI_API_KEY is not set and no .env file was found." >&2
    exit 1
  fi
fi
# Final check to ensure the API key was loaded successfully from .env
if [ -z "${XAI_API_KEY}" ]; then
  echo "## 💥💥 ERROR: .env file was found, but it does not contain XAI_API_KEY." >&2
  exit 1
fi
PROMPT=$1
echo "{\"prompt\": \"${PROMPT}\"}"  >&2
curl -s https://api.x.ai/v1/chat/completions \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${XAI_API_KEY}" \
    -d '{
      "messages": [
        {
          "role": "system",
          "content": "You are a bash shell assistant."
        },
        {
          "role": "user",
          "content": "'"$PROMPT"'"
        }
      ],
      "model": "grok-3-mini",
      "stream": false,
      "temperature": 0
    }'