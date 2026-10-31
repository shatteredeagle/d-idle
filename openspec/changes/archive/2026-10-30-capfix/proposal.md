## Why

tools/capacity.sh printed nothing and was killed under a 60 s caller timeout with a 60 s window.

## What Changes

- caller timeout rule (window + 10 s), timeout 5 and no stdin on children, WARN instead of error on a failed sample, PSI io full avg60 as a WARN-only signal
