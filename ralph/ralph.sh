#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
MAX_ITERATIONS=20

HARNESS=copilot
MODEL=gpt-5.3-codex
NUMBER_TO_LOOK_FOR=5
REPO_DIR_TO_LOOK_IN=repos/avro

escape_sed_replacement() {
  printf '%s' "$1" | sed 's/[\/&]/\\&/g'
}

render_prompt() {
  local prompt_file="$1"
  local number_to_look_for
  local repo_dir_to_look_in

  number_to_look_for="$(escape_sed_replacement "$NUMBER_TO_LOOK_FOR")"
  repo_dir_to_look_in="$(escape_sed_replacement "$REPO_DIR_TO_LOOK_IN")"

  sed \
    -e "s/NUMBER_TO_LOOK_FOR/$number_to_look_for/g" \
    -e "s/REPO_DIR_TO_LOOK_IN/$repo_dir_to_look_in/g" \
    "$prompt_file"
}

run_codex_with_prompt() {
  local prompt_file="$1"

  codex exec - \
    --dangerously-bypass-approvals-and-sandbox \
    --model "$MODEL" \
    --json \
    --skip-git-repo-check \
    -C "$REPO_DIR" \
    < <(render_prompt "$prompt_file") | yq -p=json -o=yaml '.'
}

run_copilot_with_prompt() {
  local prompt_file="$1"

  copilot \
    --available-tools bash,web_fetch,view,apply_patch \
    --disable-builtin-mcps \
    --yolo \
    --output-format json \
    --model "$MODEL" \
    -p "$(render_prompt "$prompt_file")" \
    | yq -p=json -o=yaml '.'
}

run_harness_with_prompt() {
  local prompt_file="$1"

  case "$HARNESS" in
    codex)
      run_codex_with_prompt "$prompt_file"
      ;;
    copilot)
      run_copilot_with_prompt "$prompt_file"
      ;;
    *)
      echo "Unsupported HARNESS: $HARNESS" >&2
      return 2
      ;;
  esac
}

has_first_line_marker() {
  local dir="$1"
  local marker="$2"

  find "$dir" -type f -name '*.md' -exec awk -v marker="$marker" 'FNR == 1 && $0 == marker { found = 1; exit } END { exit !found }' {} \; -print -quit | grep -q .
}

main() {
  local prompt_file

  if [[ ! -f "$SCRIPT_DIR/EXPLORE_DONE" ]]; then
    prompt_file="$SCRIPT_DIR/prompt-investigate.md"
  elif has_first_line_marker "$SCRIPT_DIR/potential_vulnerabilities" "NOT_STARTED"; then
    prompt_file="$SCRIPT_DIR/prompt-validate.md"
  elif has_first_line_marker "$SCRIPT_DIR/validation_dir" "WORKING"; then
    prompt_file="$SCRIPT_DIR/prompt-replicate.md"
  else
    echo "No Ralph work markers found."
    return 0
  fi

  echo "Running with prompt $prompt_file"

  run_harness_with_prompt "$prompt_file"
}

run_loop() {
  local count

  for ((count = 1; count <= MAX_ITERATIONS; count++)); do
    if [[ -f "$SCRIPT_DIR/STOP" ]]; then
      return 0
    fi

    echo
    echo "############################"
    echo "############################"
    echo "############################"
    printf '\n\n\n\n\n'
    echo "$count/$MAX_ITERATIONS"
    printf '\n\n\n'

    main "$@"

    printf '\n\n\n'
  done
}

run_loop "$@"
