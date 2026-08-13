# <Change title>

## Why

<One or two sentences. What breaks today, and why now.>

## What Changes

- <A new capability, a modification, or a removal.>
- **BREAKING** <Only for a change that breaks an existing caller.>

### Non-goals

<Required when the change touches more than one capability. List the work
explicitly NOT included.>

- <Out of scope, and why.>

## Behavior

<The rubric. Every later artifact is reviewed against this list, and the
adversarial review reads it directly. Write each entry so a command or an
observation can decide it.>

- B1: <observable>, confirmed by `<command>`.
- B2: <what must still hold when this lands>, confirmed by `<command>`.

## Impact

- Code: <paths>
- APIs: <surfaces>
- Dependencies: <added or removed>
- Impactful actions, each of which becomes an owner gate in tasks.md:
  <`git push`, `helm upgrade`, `openspec archive`, a migration, a delete>
- Gating signal: <feature flag, config toggle, or staged sequence, or "none">
- Judgment that stays with the owner: <what automation cannot decide>
