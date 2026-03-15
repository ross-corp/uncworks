import FilterChip from "./FilterChip";

interface FilterChipGroupProps {
  label: string;
  options: { value: string; label: string }[];
  selected: string[];
  onToggle: (value: string) => void;
  onRemove?: (value: string) => void;
}

export default function FilterChipGroup({
  label,
  options,
  selected,
  onToggle,
  onRemove,
}: FilterChipGroupProps) {
  return (
    <div className="space-y-1.5">
      <h3 className="text-[10px] font-medium uppercase tracking-widest text-muted-foreground/60 px-1">
        {label}
      </h3>
      <div className="flex flex-wrap gap-1.5">
        {options.map((opt) => (
          <FilterChip
            key={opt.value}
            label={opt.label}
            active={selected.includes(opt.value)}
            onToggle={() => onToggle(opt.value)}
            onRemove={onRemove ? () => onRemove(opt.value) : undefined}
          />
        ))}
      </div>
    </div>
  );
}
