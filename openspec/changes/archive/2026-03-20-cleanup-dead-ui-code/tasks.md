## 1. Delete Dead Files

- [x] 1.1 Delete `web/src/components/SpecEditor.tsx`
- [x] 1.2 Delete `web/src/hooks/use-mobile.tsx`
- [x] 1.3 Delete `web/src/hooks/use-toast.ts`

## 2. Verify No Breakage

- [x] 2.1 Grep for `SpecEditor`, `use-mobile`, `use-toast` imports — confirm zero results
- [x] 2.2 Run `tsc --noEmit` — confirm zero errors
