import type { Meta, StoryObj } from "@storybook/react-vite";
import Layout from "./Layout";
import { fn } from "storybook/test";

const meta: Meta<typeof Layout> = {
  title: "Components/Layout",
  component: Layout,
  args: {
    onNewRun: fn(),
    onSearchChange: fn(),
    searchQuery: "",
  },
  parameters: {
    layout: "fullscreen",
  },
};
export default meta;
type Story = StoryObj<typeof Layout>;

export const RunsView: Story = {
  args: {
    activeView: "runs",
    sidebar: (
      <div className="flex h-screen w-56 items-center justify-center border-r border-border bg-background text-xs text-muted-foreground/60">
        Sidebar
      </div>
    ),
    children: (
      <div className="flex items-center justify-center p-12 text-sm text-muted-foreground/60">
        Main content area
      </div>
    ),
  },
};

export const WithDetailPanel: Story = {
  args: {
    activeView: "runs",
    sidebar: (
      <div className="flex h-screen w-56 items-center justify-center border-r border-border bg-background text-xs text-muted-foreground/60">
        Sidebar
      </div>
    ),
    children: (
      <div className="flex items-center justify-center p-12 text-sm text-muted-foreground/60">
        Main content
      </div>
    ),
    detailPanel: (
      <div className="flex h-full items-center justify-center border-l border-border bg-background text-xs text-muted-foreground/60">
        Detail Panel
      </div>
    ),
  },
};
