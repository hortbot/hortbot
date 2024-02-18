#!/bin/sh

cd "${0%/*}"

curl -L https://unpkg.com/user-agents/src/user-agents.json.gz > user-agents.json.gz
