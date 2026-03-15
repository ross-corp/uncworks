/**
 * LiveIndicators — phosphor pulse, radar sweep, and data stream
 * background components for the orchestration graph nodes.
 */

/**
 * PhosphorPulse — wraps children with a pulsing green glow border.
 * Used on Running nodes.
 */
export function PhosphorPulse({
  active,
  children,
}: {
  active: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className={active ? "muthr-pulse" : ""}>
      {children}
    </div>
  );
}

/**
 * RadarSweep — a spinning radar indicator behind the root spec node.
 * Shows while any child agent is still running.
 */
export function RadarSweep({ active }: { active: boolean }) {
  if (!active) return null;

  return (
    <div className="absolute inset-0 flex items-center justify-center pointer-events-none overflow-hidden">
      <div
        className="muthr-radar-sweep"
        style={{
          width: "120%",
          height: "120%",
          borderRadius: "50%",
          background: `conic-gradient(
            from 0deg,
            transparent 0deg,
            rgba(0, 255, 65, 0.15) 30deg,
            transparent 60deg
          )`,
          opacity: 0.5,
        }}
      />
    </div>
  );
}

/**
 * DataStreamBackground — subtle hex waterfall effect on nodes
 * that are actively producing output.
 */
export function DataStreamBackground({
  active,
  children,
}: {
  active: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className={`relative ${active ? "muthr-data-stream" : ""}`}>
      {children}
    </div>
  );
}
