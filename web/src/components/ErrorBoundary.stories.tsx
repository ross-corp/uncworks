import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import ErrorBoundary from "./ErrorBoundary";

const meta: Meta<typeof ErrorBoundary> = {
  title: "Components/ErrorBoundary",
  component: ErrorBoundary,
};
export default meta;
type Story = StoryObj<typeof ErrorBoundary>;

function ThrowingComponent(): ReactNode {
  throw new Error("Something went wrong in the component tree");
}

export const WithError: Story = {
  render: () => (
    <ErrorBoundary>
      <ThrowingComponent />
    </ErrorBoundary>
  ),
};

export const WithChildren: Story = {
  render: () => (
    <ErrorBoundary>
      <div className="p-4 text-sm text-muted-foreground">
        This content renders normally when there is no error.
      </div>
    </ErrorBoundary>
  ),
};
