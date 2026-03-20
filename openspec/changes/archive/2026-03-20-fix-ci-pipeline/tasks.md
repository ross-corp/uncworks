## 1. Fix CI Workflow

- [x] 1.1 Rewrite .github/workflows/ci.yml with proper envtest setup, npm installs, Go build step, and caching
- [x] 1.2 Add web UI TypeScript check (npm ci + tsc --noEmit)
- [x] 1.3 Fix envtest KUBEBUILDER_ASSETS — use eval pattern not pipe to GITHUB_ENV
- [x] 1.4 Add Go module and npm caching
- [x] 1.5 Simplify to two parallel groups: build+lint, then tests
- [ ] 1.6 Verify CI passes by pushing and checking the run
