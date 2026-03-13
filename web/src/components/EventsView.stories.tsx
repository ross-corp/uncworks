import type { Meta, StoryObj } from "@storybook/react-vite";
import EventsView from "./EventsView";
import { mockRuns } from "./__fixtures__/runs";

const meta: Meta<typeof EventsView> = {
  title: "Components/EventsView",
  component: EventsView,
};
export default meta;
type Story = StoryObj<typeof EventsView>;

export const WithEvents: Story = {
  args: { runs: mockRuns },
};

export const Empty: Story = {
  args: { runs: [] },
};
