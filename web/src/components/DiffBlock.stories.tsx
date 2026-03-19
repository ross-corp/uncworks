import type { Meta, StoryObj } from "@storybook/react";
import DiffBlock from "./DiffBlock";

const meta: Meta<typeof DiffBlock> = {
  component: DiffBlock,
  title: "Components/DiffBlock",
};
export default meta;

type Story = StoryObj<typeof DiffBlock>;

export const Default: Story = {
  args: {
    content: `--- a/src/auth.ts
+++ b/src/auth.ts
@@ -1,5 +1,10 @@
-export function auth(req) {
-  if (!req.headers.authorization) {
+import jwt from 'jsonwebtoken';
+
+export function auth(req, res, next) {
+  const token = req.headers.authorization?.replace('Bearer ', '');
+  if (!token) {
+    return res.status(401).json({ error: 'Missing token' });
+  }
+  jwt.verify(token, process.env.JWT_SECRET);
   next();
 }`,
  },
};

export const AddOnly: Story = {
  args: {
    content: `+new line 1
+new line 2
+new line 3`,
  },
};
