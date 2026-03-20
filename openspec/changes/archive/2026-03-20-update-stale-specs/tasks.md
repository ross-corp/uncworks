## 1. Fix ui-theming Spec

- [ ] 1.1 Update `openspec/specs/ui-theming/spec.md` — change requirement from 12 shadcn themes to light/dark/system mode toggle
- [ ] 1.2 Verify ThemeProvider implementation matches updated spec

## 2. Fix sidecar-exec Spec

- [ ] 2.1 Update `openspec/specs/sidecar-exec/spec.md` — change path examples from `/workspace/src/` to `/workspace/<repo>/`
- [ ] 2.2 Verify sidecar `resolveWorkDir` uses the multi-repo layout

## 3. Audit All Specs

- [ ] 3.1 Grep all spec files for `/workspace/src/` and fix any remaining references
- [ ] 3.2 Spot-check 3 other specs for obvious staleness against current code
