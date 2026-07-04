#!/usr/bin/env bash
# Sanity check harness: 3 specs x N generations via Ollama, fixed config.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
MODEL="kimi-k2.7-code:cloud"
TEMP="1.0"
N=10
SYSTEM_PROMPT='You convert software specifications into Go type definitions. Output ONLY one ```go code block containing: package declaration, type definitions (structs, named types, consts) and function signatures with empty bodies {}. No explanations, no comments, no implementation logic.'

for spec in fog sharp baseline real-si real-ragivka; do
  mkdir -p "$DIR/out-cloud/$spec"
  SPEC_CONTENT="$(cat "$DIR/specs/$spec.md")"
  for i in $(seq 1 $N); do
    out="$DIR/out-cloud/$spec/$i"
    [ -s "$out.go" ] && { echo "[skip] $spec/$i"; continue; }
    echo "[gen] $spec/$i $(date +%H:%M:%S)"
    jq -n --arg model "$MODEL" --arg sys "$SYSTEM_PROMPT" --arg spec "$SPEC_CONTENT" --argjson temp "$TEMP" '{
      model: $model, stream: false, think: false,
      options: {temperature: $temp, num_predict: 2500},
      messages: [{role:"system", content:$sys}, {role:"user", content:$spec}]
    }' | curl -s --max-time 300 http://localhost:11434/api/chat -d @- \
      | jq -r '.message.content' > "$out.md"
    # Extract the go code block
    awk '/^```go/{f=1;next} /^```/{f=0} f' "$out.md" > "$out.go"
    [ -s "$out.go" ] || echo "[warn] $spec/$i: empty go extract"
    sleep 3
  done
done
echo "DONE $(date +%H:%M:%S)"
