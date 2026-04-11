import clsx from "clsx";

export function Skeleton({ className, style }: { className?: string; style?: React.CSSProperties }) {
  return <div className={clsx("skeleton", className)} style={style} />;
}

export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={clsx("space-y-2", className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          style={{ height: 14, width: i === lines - 1 ? "60%" : "100%" }}
        />
      ))}
    </div>
  );
}

export function SkeletonCard({ className }: { className?: string }) {
  return (
    <div className={clsx("card p-4 space-y-3", className)}>
      <Skeleton style={{ height: 16, width: "40%" }} />
      <SkeletonText lines={2} />
    </div>
  );
}
