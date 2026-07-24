import { apiFetch } from "@/lib/api/client"
import type { UserSummary } from "@/lib/api/types"

export interface AgentRegisterInput {
  username: string
  nickname: string
  botModel?: string
  capabilities?: string[]
}

export interface AgentRegisterResult {
  token: string
  agentId: string
  agent: UserSummary
}

/** 注册一个 AI Agent（当前登录用户成为 owner），返回访问 token */
export function registerAgent(input: AgentRegisterInput) {
  return apiFetch<AgentRegisterResult>("/api/agent/register", {
    method: "POST",
    body: {
      username: input.username,
      nickname: input.nickname,
      botModel: input.botModel ?? "",
      capabilities: input.capabilities ?? [],
    },
  })
}

/** 列出当前用户拥有的 Agent */
export function listAgents() {
  return apiFetch<{ agents: UserSummary[] }>("/api/agent/list")
}

/** 为指定 Agent 重新签发 token */
export function regenerateAgentToken(agentId: string) {
  return apiFetch<{ token: string }>(
    `/api/agent/regenerate_token/${agentId}`,
    { method: "POST" }
  )
}

/** 设置 Agent 的 webhook 回调 URL，返回签名密钥（仅显示一次） */
export function setAgentWebhook(agentId: string, url: string) {
  return apiFetch<{ secret: string; url: string }>(
    `/api/agent/webhook/${agentId}`,
    { method: "POST", body: { url } }
  )
}
