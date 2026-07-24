import { apiFetch } from "@/lib/api/client"
import type { UserSummary } from "@/lib/api/types"

export interface Skill {
  id: string
  name: string
  description: string
  installCommand: string
  tags: string[]
  rating: number
  ratingCount: number
  installCount: number
  author: UserSummary | null
  createTime: number
}

export interface SkillPage {
  results: Skill[]
  page: { page: number; limit: number; total: number }
}

export interface SkillPublishInput {
  name: string
  description?: string
  installCommand?: string
  tags?: string[]
}

/** 列出技能，可按 tag / keyword 过滤 */
export function listSkills(params?: {
  tag?: string
  keyword?: string
  page?: number
}) {
  const q = new URLSearchParams()
  if (params?.tag) q.set("tag", params.tag)
  if (params?.keyword) q.set("keyword", params.keyword)
  if (params?.page) q.set("page", String(params.page))
  const qs = q.toString()
  return apiFetch<SkillPage>(`/api/skills${qs ? `?${qs}` : ""}`)
}

/** 发布一个技能 */
export function publishSkill(input: SkillPublishInput) {
  return apiFetch<Skill>("/api/skills", {
    method: "POST",
    body: {
      name: input.name,
      description: input.description ?? "",
      installCommand: input.installCommand ?? "",
      tags: input.tags ?? [],
    },
  })
}

/** 给技能评分（1-5，每人每技能一次） */
export function rateSkill(id: string, score: number) {
  return apiFetch<null>(`/api/skills/rate/${id}`, {
    method: "POST",
    body: { score },
  })
}

/** 记录一次安装，返回安装命令 */
export function installSkill(id: string) {
  return apiFetch<{ installCommand: string }>(`/api/skills/install/${id}`, {
    method: "POST",
  })
}
