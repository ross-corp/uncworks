import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "./ui/dialog";
import { Button } from "./ui/button";

interface HitlModalProps {
  open: boolean;
  promptText: string;
  onSubmit: (input: string) => void;
  onClose: () => void;
}

export default function HitlModal({ open, promptText, onSubmit, onClose }: HitlModalProps) {
  const [value, setValue] = useState("");

  function handleSubmit() {
    if (!value.trim()) return;
    onSubmit(value.trim());
    setValue("");
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      handleSubmit();
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent showCloseButton>
        <DialogHeader>
          <DialogTitle>Agent is waiting for input</DialogTitle>
          {promptText && (
            <DialogDescription className="whitespace-pre-wrap text-xs font-mono mt-1">
              {promptText}
            </DialogDescription>
          )}
        </DialogHeader>
        <textarea
          autoFocus
          className="w-full border bg-background px-3 py-2 text-sm outline-none focus:border-primary min-h-[80px] resize-none font-mono"
          placeholder="Type your response..."
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
        />
        <DialogFooter>
          <Button variant="outline" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button size="sm" onClick={handleSubmit} disabled={!value.trim()}>
            Send
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
