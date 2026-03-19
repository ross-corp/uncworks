import type { Meta, StoryObj } from "@storybook/react";
import RunStatusBadge from "./RunStatusBadge";

const meta: Meta<typeof RunStatusBadge> = {
  component: RunStatusBadge,
  title: "Components/RunStatusBadge",
};
export default meta;

type Story = StoryObj<typeof RunStatusBadge>;

export const Running: Story = { args: { phase: "running" } };
export const Succeeded: Story = { args: { phase: "succeeded" } };
export const Failed: Story = { args: { phase: "failed" } };
export const Pending: Story = { args: { phase: "pending" } };
export const WaitingForInput: Story = { args: { phase: "waiting_for_input" } };
export const Cancelled: Story = { args: { phase: "cancelled" } };
