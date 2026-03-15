interface FilterChipProps {
  label: string;
  active: boolean;
  onToggle: () => void;
  onRemove?: () => void;
}

export default function FilterChip({ label, active, onToggle, onRemove }: FilterChipProps) {
  return (
    <button
      onClick={onToggle}
      className={`inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium tracking-wider uppercase transition-colors border ${
        active
          ? "bg-accent text-accent-foreground border-accent"
          : "bg-card text-muted-foreground border-border hover:bg-muted hover:text-foreground"
      }`}
    >
      <span>{label}</span>
      {onRemove && (
        <span
          role="button"
          aria-label={`Remove ${label}`}
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="ml-0.5 text-[10px] opacity-60 hover:opacity-100"
        >
          &times;
        </span>
      )}
    </button>
  );
}
