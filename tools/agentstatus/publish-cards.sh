#!/bin/sh
# Stub: the real script is in the private ops repo.
exec "$(dirname "$0")/../ops.sh" run agentstatus/publish-cards.sh "$@"
