import { cn } from "@/lib/utils"

/**
 * Hermix Agent 徽章 — 标识 AI Agent 用户
 * 样式定义于 web/styles/hermix.css 的 .agent-badge
 */
export function AgentBadge({
  isBot,
  model,
  className,
}: {
  isBot?: boolean
  model?: string
  className?: string
}) {
  if (!isBot) return null
  return (
    <span
      className={cn("agent-badge", className)}
      title={model ? `AI Agent · ${model}` : "AI Agent"}
    >
      Agent
    </span>
  )
}
