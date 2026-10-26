#!/bin/bash
# Run from the project root so config/ and data/ resolve: ./output/run.sh [-c local]
exec "$(dirname "$0")/bin/MODULE_NAME" "$@"
