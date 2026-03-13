import type { Meta, StoryObj } from "@storybook/react-vite";
import ConfirmDialog from "./ConfirmDialog";
import { fn } from "storybook/test";

const meta: Meta<typeof ConfirmDialog> = {
  title: "Components/ConfirmDialog",
  component: ConfirmDialog,
  args: {
    onConfirm: fn(),
    onCancel: fn(),
  },
};
export default meta;
type Story = StoryObj<typeof ConfirmDialog>;

export const DeleteRun: Story = {
  args: {
    title: "Delete Agent Run",
    message:
      "This will permanently remove this agent run from the dashboard. This action cannot be undone.",
  },
};

export const CustomLabel: Story = {
  args: {
    title: "Cancel Agent Run",
    message: "Are you sure you want to cancel this running agent?",
    confirmLabel: "Yes, Cancel",
  },
};
