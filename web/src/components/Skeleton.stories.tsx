import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton, SkeletonRow, SkeletonDetail } from "./Skeleton";

const meta: Meta<typeof Skeleton> = {
  title: "Components/Skeleton",
  component: Skeleton,
};
export default meta;
type Story = StoryObj<typeof Skeleton>;

export const Default: Story = {
  args: { className: "h-4 w-48" },
};

export const TableRows: StoryObj = {
  render: () => (
    <table className="w-full text-sm" style={{ tableLayout: "fixed" }}>
      <tbody>
        <SkeletonRow />
        <SkeletonRow />
        <SkeletonRow />
      </tbody>
    </table>
  ),
};

export const DetailPanel: StoryObj = {
  render: () => (
    <div style={{ width: 480, height: "100vh" }}>
      <SkeletonDetail />
    </div>
  ),
};
