import type { Meta, StoryObj } from "@storybook/react-vite";
import Sidebar from "./Sidebar";
import { mockRuns } from "./__fixtures__/runs";
import { fn } from "storybook/test";
import type { Workspace } from "../hooks/useWorkspaces";

const mockWorkspaces: Workspace[] = [
  {
    id: "ws-1",
    name: "backend",
    description: "Backend services",
    repos: [{ url: "https://github.com/acme/backend.git", branch: "main" }],
  },
  {
    id: "ws-2",
    name: "frontend",
    description: "Frontend apps",
    repos: [{ url: "https://github.com/acme/frontend.git", branch: "main" }],
  },
];

const meta: Meta<typeof Sidebar> = {
  title: "Components/Sidebar",
  component: Sidebar,
  args: {
    onSelectRepo: fn(),
    onPhaseFilter: fn(),
    onOpenRepos: fn(),
    onOpenEvents: fn(),
    onSelectWorkspace: fn(),
    onNewWorkspace: fn(),
    onEditWorkspace: fn(),
    workspaces: mockWorkspaces,
    selectedWorkspace: null,
  },
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <div style={{ width: 224, height: "100vh" }}>
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof Sidebar>;

export const Default: Story = {
  args: {
    runs: mockRuns,
    repos: [
      "https://github.com/acme/backend.git",
      "https://github.com/acme/frontend.git",
      "https://github.com/acme/infra.git",
    ],
    selectedRepo: null,
    phaseFilter: "all",
  },
};

export const RepoSelected: Story = {
  args: {
    runs: mockRuns,
    repos: [
      "https://github.com/acme/backend.git",
      "https://github.com/acme/frontend.git",
    ],
    selectedRepo: "https://github.com/acme/backend.git",
    phaseFilter: "all",
  },
};

export const PhaseFiltered: Story = {
  args: {
    runs: mockRuns,
    repos: ["https://github.com/acme/backend.git"],
    selectedRepo: null,
    phaseFilter: "running",
  },
};

export const WorkspaceSelected: Story = {
  args: {
    runs: mockRuns,
    repos: ["https://github.com/acme/backend.git"],
    selectedRepo: null,
    phaseFilter: "all",
    selectedWorkspace: "backend",
  },
};
