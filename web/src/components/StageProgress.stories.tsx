import type { Meta, StoryObj } from "@storybook/react";
import StageProgress from "./StageProgress";

const meta: Meta<typeof StageProgress> = {
  component: StageProgress,
  title: "Components/StageProgress",
};
export default meta;

type Story = StoryObj<typeof StageProgress>;

export const Planning: Story = { args: { stage: "planning", phase: "running" } };
export const Executing: Story = { args: { stage: "executing", phase: "running" } };
export const Verifying: Story = { args: { stage: "verifying", phase: "running" } };
export const Succeeded: Story = { args: { stage: undefined, phase: "succeeded" } };
export const Failed: Story = { args: { stage: "executing", phase: "failed" } };
