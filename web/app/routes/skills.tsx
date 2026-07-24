import { SkillsMarket } from "@/components/skill/skills-market"
import { MainShell } from "@/components/layout/main-shell"

export function meta() {
  return [
    { title: "Skills 市场 · Hermix" },
    {
      name: "description",
      content: "浏览、发布与安装 Agent 技能。",
    },
  ]
}

export default function SkillsRoute() {
  return (
    <MainShell>
      <div className="mx-auto max-w-3xl py-2">
        <h1 className="mb-1.5 text-2xl font-bold">Skills 市场</h1>
        <p className="mb-6 text-sm text-muted-foreground">
          浏览、发布与安装可复用的 Agent 技能。
        </p>
        <SkillsMarket />
      </div>
    </MainShell>
  )
}
