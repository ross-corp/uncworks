## MODIFIED Requirements

### Requirement: Namespace is autodetected from cluster, not manually entered
The system SHALL autodetect the UNCWORKS namespace by listing namespaces in the active kubecontext and selecting the one matching `uncworks` or labelled `app.kubernetes.io/part-of=uncworks`. Manual namespace entry SHALL only be shown if autodetection finds zero or multiple candidates.

#### Scenario: Single matching namespace found
- **WHEN** wizard step 1 runs namespace detection and finds exactly one matching namespace
- **THEN** that namespace is auto-selected and shown to the user as read-only with a "Detected" badge

#### Scenario: No matching namespace found
- **WHEN** wizard step 1 finds no namespace matching the autodetection criteria
- **THEN** a text input is shown for manual namespace entry with a placeholder of `uncworks`

#### Scenario: Multiple matching namespaces found
- **WHEN** wizard step 1 finds more than one namespace matching the criteria
- **THEN** a dropdown is shown listing all candidates for the user to select

### Requirement: Kubecontext is selectable in wizard step 1
The system SHALL list all kubecontexts from `~/.kube/config` and show the active context pre-selected. The user SHALL be able to change the context before proceeding.

#### Scenario: Active context pre-selected
- **WHEN** wizard step 1 is shown
- **THEN** the kubecontext dropdown shows the current active context as the selected value

#### Scenario: Context changed in wizard
- **WHEN** the user selects a different kubecontext in the dropdown
- **THEN** namespace autodetection reruns against the newly selected context
