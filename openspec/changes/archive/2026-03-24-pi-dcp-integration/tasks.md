## 1. Install pi-dcp in Sidecar Image

- [x] 1.1 Add pi-dcp clone and npm install to `docker/Dockerfile.sidecar` — clone into `/root/.pi/agent/extensions/pi-dcp` with `--depth 1`, run `npm install`
- [ ] 1.2 Verify pi-dcp initializes when pi agent starts inside a container built from the updated image

## 2. Sidecar DCP Event Detection (DONE)

- [x] 2.1 Add `piDcpPrunedRe` regex to `internal/sidecar/gateway.go` to match `[pi-dcp] Pruned N / M messages` lines
- [x] 2.2 Add DCP log line detection in the plain-text fallback block of the stdout parser
- [x] 2.3 Create compaction span when pruning summary is matched — instant span with `source: "pi-dcp"` metadata
- [x] 2.4 Add `parsePiDcpPruned` function to extract pruned and total counts

## 3. Frontend (DONE — reuses compaction type)

- [x] 3.1 DCP events render as compaction spans with orange/amber styling in TraceTimeline
- [x] 3.2 Inline label shows token reduction from metadata
- [x] 3.3 Detail panel shows full compaction metadata including source

## 4. Verification

- [ ] 4.1 Run a test with pi-dcp enabled and verify DCP pruning spans appear in the trace timeline
- [ ] 4.2 Verify pruning stats (pruned/total counts) are correctly displayed
