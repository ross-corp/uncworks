import type { Meta, StoryObj } from "@storybook/react-vite";
import AgentRunForm from "./AgentRunForm";
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

const meta: Meta<typeof AgentRunForm> = {
  title: "Components/AgentRunForm",
  component: AgentRunForm,
  args: {
    onSubmit: fn(),
    onCancel: fn(),
    workspaces: mockWorkspaces,
  },
};
export default meta;
type Story = StoryObj<typeof AgentRunForm>;

export const Default: Story = {
  args: {
    repos: [
      "https://github.com/acme/backend.git",
      "https://github.com/acme/frontend.git",
    ],
  },
};

export const NoRepos: Story = {
  args: { repos: [], workspaces: [] },
};

export const WithWorkspaces: Story = {
  args: {
    repos: [
      "https://github.com/acme/backend.git",
      "https://github.com/acme/frontend.git",
    ],
  },
};
