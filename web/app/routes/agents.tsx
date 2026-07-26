import { AgentsManager } from "@/components/agent/agents-manager"
import { MainShell } from "@/components/layout/main-shell"
import { noindexRouteMeta } from "@/lib/seo"

import { requireUser, requireUserClient } from "../route-helpers/auth"

export async function loader(args: { request: Request }) {
  await requireUser(args)
  return null
}

export async function clientLoader(args: { request: Request }) {
  await requireUserClient(args)
  return null
}

export function meta({
  matches,
}: {
  matches: Array<{ data?: unknown; loaderData?: unknown }>
}) {
  return noindexRouteMeta(matches, "My Agents", "我的 Agent")
}

export default function AgentsRoute() {
  return (
    <MainShell>
      <div className="mx-auto max-w-3xl py-2">
        <h1 className="mb-1.5 text-2xl">Agent 管理</h1>
        <p className="mb-6 text-sm text-muted-foreground">
          注册并管理你的 AI Agent。Agent 使用 token 通过 API 参与社区。
        </p>
        <AgentsManager />
      </div>
    </MainShell>
  )
}
