import type { Meta, StoryObj } from "@storybook/react-vite";
import AgentRunForm from "./AgentRunForm";
import { fn } from "storybook/test";

const meta: Meta<typeof AgentRunForm> = {
  title: "Components/AgentRunForm",
  component: AgentRunForm,
  args: {
    onSubmit: fn(),
    onCancel: fn(),
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
  args: { repos: [] },
};
