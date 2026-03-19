import ReactMarkdown from "react-markdown";

interface ChatMessageProps {
  role: "user" | "agent" | "system";
  content: string;
  model?: string;
}

const wrapperStyles: Record<ChatMessageProps["role"], string> = {
  user: "flex justify-end",
  agent: "flex justify-start",
  system: "flex justify-center",
};

const bubbleStyles: Record<ChatMessageProps["role"], string> = {
  user: "max-w-[75%] rounded-lg px-4 py-2 bg-blue-600/20 text-foreground",
  agent: "max-w-[75%] rounded-lg px-4 py-2 bg-muted text-foreground",
  system: "max-w-[85%] px-4 py-1.5 text-muted-foreground italic text-sm text-center",
};

export default function ChatMessage({ role, content, model }: ChatMessageProps) {
  return (
    <div className={wrapperStyles[role]}>
      <div className={bubbleStyles[role]}>
        {role === "agent" && model && (
          <span className="mb-1 inline-block rounded bg-muted-foreground/15 px-1.5 py-0.5 text-[10px] text-muted-foreground">
            {model}
          </span>
        )}
        <div className="prose prose-sm prose-invert max-w-none break-words">
          <ReactMarkdown>{content}</ReactMarkdown>
        </div>
      </div>
    </div>
  );
}
