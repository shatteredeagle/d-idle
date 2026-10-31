#!/bin/sh
# Stub: the status board CLI is in the private ops repo.
exec "$(dirname "$0")/../ops.sh" run agentstatus/agentstatus.py "$@"
