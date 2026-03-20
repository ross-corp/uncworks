## Purpose

Ensure OpenSpec specs accurately reflect the current implementation so that agent verification produces correct results.

## ADDED Requirements

### Requirement: Theme support matches implementation
The ui-theming spec SHALL require light mode, dark mode, and a system-preference toggle instead of 12 shadcn themes.

#### Scenario: Light/dark mode toggle
- **WHEN** the user toggles the theme
- **THEN** the app switches between light and dark mode via `class="dark"` on the root element

#### Scenario: System preference
- **WHEN** the user selects "system" mode
- **THEN** the app follows the OS dark/light preference

### Requirement: Workspace paths use multi-repo layout
The sidecar-exec spec SHALL use `/workspace/<repo>/` as the canonical path instead of `/workspace/src/`.

#### Scenario: Single repo run
- **WHEN** an agent run targets repo `my-app`
- **THEN** the working directory is `/workspace/my-app/`

#### Scenario: No stale path references
- **WHEN** grepping all spec files for `/workspace/src/`
- **THEN** zero matches are returned
