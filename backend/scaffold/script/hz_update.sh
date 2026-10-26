#!/bin/bash
# Regenerates biz/router, biz/model and new handler stubs from idl/. Existing
# handler bodies and biz/router/*/middleware.go are preserved.
set -euo pipefail
cd "$(dirname "$0")/.."
hz update -idl idl/api.thrift -handler_dir biz/handler -model_dir biz/model
hz update -idl idl/papi.thrift -handler_dir biz/handler -model_dir biz/model
