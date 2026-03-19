import type { Meta, StoryObj } from "@storybook/react";
import ChatMessage from "./ChatMessage";

const meta: Meta<typeof ChatMessage> = {
  component: ChatMessage,
  title: "Components/ChatMessage",
};
export default meta;

type Story = StoryObj<typeof ChatMessage>;

export const User: Story = {
  args: { role: "user", content: "Fix the auth middleware to validate JWT tokens" },
};

export const Agent: Story = {
  args: {
    role: "agent",
    content: "I'll add JWT validation using `jsonwebtoken`. Here's the plan:\n\n1. Parse the Bearer token\n2. Verify against JWT_SECRET\n3. Return 401 for invalid tokens",
    model: "qwen3:8b",
  },
};

export const System: Story = {
  args: { role: "system", content: "Agent started" },
};
