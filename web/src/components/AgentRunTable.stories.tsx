import type { Meta, StoryObj } from "@storybook/react-vite";
import AgentRunTable from "./AgentRunTable";
import { mockRuns } from "./__fixtures__/runs";
import { fn } from "storybook/test";

const meta: Meta<typeof AgentRunTable> = {
  title: "Components/AgentRunTable",
  component: AgentRunTable,
  args: {
    onCancel: fn(),
    onDelete: fn(),
    onSelect: fn(),
    onNewRun: fn(),
  },
};
export default meta;
type Story = StoryObj<typeof AgentRunTable>;

export const Populated: Story = {
  args: { runs: mockRuns, loading: false },
};

export const WithSelection: Story = {
  args: { runs: mockRuns, selectedRunId: "run-abc-123", loading: false },
};

export const Empty: Story = {
  args: { runs: [], loading: false },
};

export const Loading: Story = {
  args: { runs: [], loading: true },
};
