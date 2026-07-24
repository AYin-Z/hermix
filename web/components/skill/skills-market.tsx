import * as React from "react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  listSkills,
  publishSkill,
  rateSkill,
  installSkill,
  type Skill,
} from "@/lib/api/skills"
import { msgError, msgSuccess } from "@/lib/toast"

export function SkillsMarket() {
  const [skills, setSkills] = React.useState<Skill[]>([])
  const [loading, setLoading] = React.useState(true)
  const [keyword, setKeyword] = React.useState("")

  const reload = React.useCallback((kw?: string) => {
    setLoading(true)
    listSkills({ keyword: kw })
      .then((res) => setSkills(res.results ?? []))
      .catch((e) => msgError(e?.message || "加载失败"))
      .finally(() => setLoading(false))
  }, [])

  React.useEffect(() => {
    reload()
  }, [reload])

  return (
    <div className="space-y-6">
      <PublishForm onDone={() => reload(keyword)} />
      <div className="flex items-center gap-2">
        <Input
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder="搜索技能名称…"
          className="max-w-xs"
          onKeyDown={(e) => e.key === "Enter" && reload(keyword)}
        />
        <Button variant="outline" onClick={() => reload(keyword)}>
          搜索
        </Button>
      </div>
      <SkillList skills={skills} loading={loading} onChanged={() => reload(keyword)} />
    </div>
  )
}

function PublishForm({ onDone }: { onDone: () => void }) {
  const [name, setName] = React.useState("")
  const [description, setDescription] = React.useState("")
  const [installCommand, setInstallCommand] = React.useState("")
  const [tags, setTags] = React.useState("")
  const [busy, setBusy] = React.useState(false)

  async function submit() {
    if (!name.trim()) {
      msgError("技能名称必填")
      return
    }
    setBusy(true)
    try {
      await publishSkill({
        name: name.trim(),
        description: description.trim(),
        installCommand: installCommand.trim(),
        tags: tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
      })
      msgSuccess("技能已发布")
      setName("")
      setDescription("")
      setInstallCommand("")
      setTags("")
      onDone()
    } catch (e) {
      msgError(e instanceof Error ? e.message : "发布失败")
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card>
      <CardContent className="space-y-3 py-4">
        <h2 className="font-semibold">发布技能</h2>
        <div className="grid gap-3 sm:grid-cols-2">
          <div>
            <Label htmlFor="sk-name">名称</Label>
            <Input id="sk-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <Label htmlFor="sk-tags">标签（逗号分隔）</Label>
            <Input id="sk-tags" value={tags} onChange={(e) => setTags(e.target.value)} placeholder="code-review, qa" />
          </div>
        </div>
        <div>
          <Label htmlFor="sk-desc">描述</Label>
          <Input id="sk-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <div>
          <Label htmlFor="sk-cmd">安装命令</Label>
          <Input id="sk-cmd" value={installCommand} onChange={(e) => setInstallCommand(e.target.value)} placeholder="hermix install my-skill" />
        </div>
        <Button onClick={submit} disabled={busy}>
          {busy ? "发布中…" : "发布"}
        </Button>
      </CardContent>
    </Card>
  )
}

function SkillList({
  skills,
  loading,
  onChanged,
}: {
  skills: Skill[]
  loading: boolean
  onChanged: () => void
}) {
  if (loading) {
    return <p className="text-sm text-muted-foreground">加载中…</p>
  }
  if (skills.length === 0) {
    return <p className="text-sm text-muted-foreground">还没有技能，发布第一个吧。</p>
  }
  return (
    <div className="grid gap-3">
      {skills.map((s) => (
        <SkillCard key={s.id} skill={s} onChanged={onChanged} />
      ))}
    </div>
  )
}

function SkillCard({ skill, onChanged }: { skill: Skill; onChanged: () => void }) {
  async function onRate(score: number) {
    try {
      await rateSkill(skill.id, score)
      msgSuccess("评分成功")
      onChanged()
    } catch (e) {
      msgError(e instanceof Error ? e.message : "评分失败")
    }
  }
  async function onInstall() {
    try {
      const res = await installSkill(skill.id)
      await navigator.clipboard?.writeText(res.installCommand || "")
      msgSuccess("安装命令已复制到剪贴板")
      onChanged()
    } catch (e) {
      msgError(e instanceof Error ? e.message : "操作失败")
    }
  }
  return (
    <Card>
      <CardContent className="flex flex-wrap items-start justify-between gap-3 py-4">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-medium">{skill.name}</span>
            <span className="text-xs text-muted-foreground">
              ★ {skill.rating.toFixed(1)} ({skill.ratingCount}) · 安装 {skill.installCount}
            </span>
          </div>
          {skill.description ? (
            <p className="mt-1 text-sm text-muted-foreground">{skill.description}</p>
          ) : null}
          {skill.tags.length > 0 ? (
            <div className="mt-1.5 flex flex-wrap gap-1">
              {skill.tags.map((t) => (
                <span key={t} className="rounded-sm bg-accent px-1.5 py-0.5 text-xs text-accent-foreground">
                  {t}
                </span>
              ))}
            </div>
          ) : null}
          <p className="mt-1 text-xs text-muted-foreground">
            by {skill.author?.nickname ?? "未知"}
          </p>
        </div>
        <div className="flex items-center gap-1.5">
          <select
            className="h-8 rounded-md border bg-background px-2 text-xs"
            defaultValue=""
            onChange={(e) => e.target.value && onRate(Number(e.target.value))}
          >
            <option value="" disabled>
              评分
            </option>
            {[5, 4, 3, 2, 1].map((n) => (
              <option key={n} value={n}>
                {n} 星
              </option>
            ))}
          </select>
          <Button variant="outline" size="sm" onClick={onInstall}>
            安装
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
