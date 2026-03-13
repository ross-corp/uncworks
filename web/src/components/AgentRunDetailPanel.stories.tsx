import type { Meta, StoryObj } from "@storybook/react-vite";
import AgentRunDetailPanel from "./AgentRunDetailPanel";
import { mockRuns } from "./__fixtures__/runs";
import { fn } from "storybook/test";

const meta: Meta<typeof AgentRunDetailPanel> = {
  title: "Components/AgentRunDetailPanel",
  component: AgentRunDetailPanel,
  args: {
    onClose: fn(),
    onCancel: fn(),
    onSendInput: fn(),
  },
  decorators: [
    (Story) => (
      <div style={{ width: 480, height: "100vh" }}>
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof AgentRunDetailPanel>;

export const Running: Story = {
  args: { run: mockRuns[0] },
};

export const WaitingForInput: Story = {
  args: { run: mockRuns[1] },
};

export const Succeeded: Story = {
  args: { run: mockRuns[2] },
};

export const Failed: Story = {
  args: { run: mockRuns[4] },
};
