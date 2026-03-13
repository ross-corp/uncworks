import type { Meta, StoryObj } from "@storybook/react-vite";
import { PhaseBadge, BackendBadge, ModelTierBadge } from "./StatusBadge";

const phaseMeta: Meta<typeof PhaseBadge> = {
  title: "Components/StatusBadge/PhaseBadge",
  component: PhaseBadge,
};
export default phaseMeta;

type PhaseStory = StoryObj<typeof PhaseBadge>;

export const Pending: PhaseStory = { args: { phase: "pending" } };
export const Running: PhaseStory = { args: { phase: "running" } };
export const WaitingForInput: PhaseStory = { args: { phase: "waiting_for_input" } };
export const Succeeded: PhaseStory = { args: { phase: "succeeded" } };
export const Failed: PhaseStory = { args: { phase: "failed" } };
export const Cancelled: PhaseStory = { args: { phase: "cancelled" } };

export const AllPhases: StoryObj = {
  render: () => (
    <div className="flex flex-wrap gap-2 p-4">
      <PhaseBadge phase="pending" />
      <PhaseBadge phase="running" />
      <PhaseBadge phase="waiting_for_input" />
      <PhaseBadge phase="succeeded" />
      <PhaseBadge phase="failed" />
      <PhaseBadge phase="cancelled" />
    </div>
  ),
};

export const AllBackends: StoryObj = {
  render: () => (
    <div className="flex flex-wrap gap-2 p-4">
      <BackendBadge backend="pod" />
      <BackendBadge backend="kubevirt" />
      <BackendBadge backend="external" />
    </div>
  ),
};

export const AllModelTiers: StoryObj = {
  render: () => (
    <div className="flex flex-wrap gap-2 p-4">
      <ModelTierBadge tier="default" />
      <ModelTierBadge tier="default-cloud" />
      <ModelTierBadge tier="premium" />
    </div>
  ),
};
