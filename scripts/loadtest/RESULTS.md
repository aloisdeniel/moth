# Load-test results

Paste measured `ghz` summaries here, each under a heading naming the exact
hardware, moth version, and argon2 parameters it ran with. Only measured
numbers belong in this file — see README.md.

## Template

```
### <cpu> · <ram> · moth <version> · argon2(<m>,<t>,<p>)

Command: MOTH_PK=… CONCURRENCY=50 TOTAL=5000 ./signin.sh
Summary:
  Count:        5000
  Total:        …
  Requests/sec: …
  Latency  p50: …   p95: …   p99: …
  Status OK:    5000 / 5000
```

<!-- No runs recorded yet. The README's baseline stays "measure it yourself"
     until a real run is pasted above. -->
