import type { Meta, StoryObj } from "@storybook/react-vite";
import Sidebar from "./Sidebar";
import { mockRuns } from "./__fixtures__/runs";
import { fn } from "storybook/test";

const meta: Meta<typeof Sidebar> = {
  title: "Components/Sidebar",
  component: Sidebar,
  args: {
    onSelectRepo: fn(),
    onPhaseFilter: fn(),
    onOpenRepos: fn(),
    onOpenEvents: fn(),
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
