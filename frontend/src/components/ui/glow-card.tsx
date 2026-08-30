import * as React from "react"
import { cn } from "@/lib/utils"

/**
 * Spotlight/glow card: radial glow follows mouse cursor.
 * Cult-inspired aesthetic, self-contained (no extra deps).
 */

interface GlowCardProps extends React.HTMLAttributes<HTMLDivElement> {
  glowColor?: string
  innerClassName?: string
}

export function GlowCard({
  className,
  innerClassName,
  glowColor = "rgba(245,197,66,0.40)",
  children,
  onMouseMove,
  onMouseLeave,
  ...props
}: GlowCardProps) {
  const ref = React.useRef<HTMLDivElement>(null)
  const [pos, setPos] = React.useState({ x: 0, y: 0 })
  const [active, setActive] = React.useState(false)

  const handleMove = React.useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      const rect = (e.currentTarget as HTMLDivElement).getBoundingClientRect()
      setPos({ x: e.clientX - rect.left, y: e.clientY - rect.top })
      setActive(true)
      onMouseMove?.(e)
    },
    [onMouseMove]
  )

  const handleLeave = React.useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      setActive(false)
      onMouseLeave?.(e)
    },
    [onMouseLeave]
  )

  return (
    <div
      ref={ref}
      onMouseMove={handleMove}
      onMouseLeave={handleLeave}
      className={cn(
        "relative overflow-hidden rounded-xl border border-border bg-card shadow-sm transition-colors hover:border-primary/30",
        className
      )}
      {...props}
    >
      <div
        className="pointer-events-none absolute inset-0 transition-opacity duration-300"
        style={{
          opacity: active ? 1 : 0,
          background: `radial-gradient(250px circle at ${pos.x}px ${pos.y}px, ${glowColor}, transparent 70%)`,
        }}
      />
      <div className={cn("relative z-10", innerClassName)}>{children}</div>
    </div>
  )
}