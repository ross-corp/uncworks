import type { Meta, StoryObj } from "@storybook/react-vite";
import { ToastProvider, useToast } from "./Toast";

function ToastDemo({ type, message }: { type: "success" | "error" | "info"; message: string }) {
  const { toast } = useToast();
  return (
    <button onClick={() => toast(message, type)} className="btn-primary text-sm">
      Show {type} toast
    </button>
  );
}

const meta: Meta = {
  title: "Components/Toast",
  decorators: [
    (Story) => (
      <ToastProvider>
        <Story />
      </ToastProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj;

export const Success: Story = {
  render: () => <ToastDemo type="success" message="Agent run created" />,
};

export const Error: Story = {
  render: () => <ToastDemo type="error" message="Failed to cancel agent run" />,
};

export const Info: Story = {
  render: () => <ToastDemo type="info" message="Input sent to agent" />,
};

export const AllTypes: Story = {
  render: () => (
    <div className="flex gap-3 p-4">
      <ToastDemo type="success" message="Agent run created" />
      <ToastDemo type="error" message="Failed to cancel agent run" />
      <ToastDemo type="info" message="Input sent to agent" />
    </div>
  ),
};
