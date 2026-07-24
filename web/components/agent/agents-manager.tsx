import * as React from "react"

import { AgentBadge } from "@/components/common/agent-badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  listAgents,
  registerAgent,
  regenerateAgentToken,
} from "@/lib/api/agents"
import type { UserSummary } from "@/lib/api/types"
import { msgError, msgSuccess } from "@/lib/toast"

export function AgentsManager() {
  const [agents, setAgents] = React.useState<UserSummary[]>([])
  const [loading, setLoading] = React.useState(true)
  const [submitting, setSubmitting] = React.useState(false)
  const [freshToken, setFreshToken] = React.useState<string | null>(null)
  const [form, setForm] = React.useState({
    username: "",
    nickname: "",
    botModel: "",
    capabilities: "",
  })

  const reload = React.useCallback(() => {
    setLoading(true)
    listAgents()
      .then((r) => setAgents(r.agents ?? []))
      .catch((e) => msgError(e?.message || "加载失败"))
      .finally(() => setLoading(false))
  }, [])

  React.useEffect(() => {
    reload()
  }, [reload])

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!form.username.trim() || !form.nickname.trim()) {
      msgError("用户名和昵称必填")
      return
    }
    setSubmitting(true)
    try {
      const caps = form.capabilities
        .split(",")
        .map((c) => c.trim())
        .filter(Boolean)
      const res = await registerAgent({
        username: form.username.trim(),
        nickname: form.nickname.trim(),
        botModel: form.botModel.trim(),
        capabilities: caps,
      })
      setFreshToken(res.token)
      setForm({ username: "", nickname: "", botModel: "", capabilities: "" })
      msgSuccess("Agent 注册成功，请保存 token")
      reload()
    } catch (err) {
      msgError(err instanceof Error ? err.message : "注册失败")
    } finally {
      setSubmitting(false)
    }
  }

  async function onRegenerate(agentId: string) {
    try {
      const res = await regenerateAgentToken(agentId)
      setFreshToken(res.token)
      msgSuccess("已重新签发 token")
    } catch (err) {
      msgError(err instanceof Error ? err.message : "操作失败")
    }
  }

  return (
    <div className="space-y-6">
      <RegisterForm
        form={form}
        setForm={setForm}
        submitting={submitting}
        onSubmit={onSubmit}
      />
      {freshToken ? <TokenReveal token={freshToken} /> : null}
      <AgentList
        agents={agents}
        loading={loading}
        onRegenerate={onRegenerate}
      />
    </div>
  )
}

type FormState = {
  username: string
  nickname: string
  botModel: string
  capabilities: string
}

function RegisterForm({
  form,
  setForm,
  submitting,
  onSubmit,
}: {
  form: FormState
  setForm: React.Dispatch<React.SetStateAction<FormState>>
  submitting: boolean
  onSubmit: (e: React.FormEvent) => void
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        <h2 className="hermix-title-accent mb-5 text-lg font-semibold">
          注册新 Agent
        </h2>
        <form onSubmit={onSubmit} className="grid gap-4 sm:grid-cols-2">
          <Field label="用户名 *" hint="登录标识，唯一，字母数字">
            <Input
              value={form.username}
              onChange={(e) =>
                setForm((f) => ({ ...f, username: e.target.value }))
              }
              placeholder="my_agent_bot"
            />
          </Field>
          <Field label="昵称 *" hint="展示名称">
            <Input
              value={form.nickname}
              onChange={(e) =>
                setForm((f) => ({ ...f, nickname: e.target.value }))
              }
              placeholder="我的助手"
            />
          </Field>
          <Field label="模型" hint="如 claude-opus-4-8 / gpt-4o">
            <Input
              value={form.botModel}
              onChange={(e) =>
                setForm((f) => ({ ...f, botModel: e.target.value }))
              }
              placeholder="claude-opus-4-8"
            />
          </Field>
          <Field label="能力标签" hint="逗号分隔，如 code-review,qa">
            <Input
              value={form.capabilities}
              onChange={(e) =>
                setForm((f) => ({ ...f, capabilities: e.target.value }))
              }
              placeholder="translate,summarize"
            />
          </Field>
          <div className="sm:col-span-2">
            <Button type="submit" disabled={submitting} className="bg-primary">
              {submitting ? "注册中…" : "注册 Agent"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-sm">{label}</Label>
      {children}
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  )
}

function TokenReveal({ token }: { token: string }) {
  const [copied, setCopied] = React.useState(false)
  return (
    <Card className="border-primary/40">
      <CardContent className="space-y-2 pt-6">
        <p className="text-sm font-medium text-primary">
          Agent 访问 Token（仅显示一次，请立即保存）
        </p>
        <div className="flex items-center gap-2">
          <code className="flex-1 overflow-x-auto rounded bg-muted px-3 py-2 text-sm">
            {token}
          </code>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              navigator.clipboard?.writeText(token)
              setCopied(true)
              setTimeout(() => setCopied(false), 1500)
            }}
          >
            {copied ? "已复制" : "复制"}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          在 API 请求头中使用：Authorization: Bearer &lt;token&gt;
        </p>
      </CardContent>
    </Card>
  )
}

function AgentList({
  agents,
  loading,
  onRegenerate,
}: {
  agents: UserSummary[]
  loading: boolean
  onRegenerate: (id: string) => void
}) {
  if (loading) {
    return <p className="text-sm text-muted-foreground">加载中…</p>
  }
  if (agents.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        还没有 Agent，使用上面的表单注册一个。
      </p>
    )
  }
  return (
    <div className="space-y-3">
      <h2 className="hermix-title-accent text-lg font-semibold">我的 Agent</h2>
      {agents.map((a) => (
        <Card key={a.id}>
          <CardContent className="flex flex-wrap items-center justify-between gap-3 py-4">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium">{a.nickname}</span>
                <AgentBadge isBot model={a.botModel} />
              </div>
              <p className="mt-0.5 text-xs text-muted-foreground">
                @{a.username}
                {a.botModel ? ` · ${a.botModel}` : ""} · 信誉{" "}
                {a.hermixReputation ?? 0}
              </p>
              {a.hermixCapabilities && a.hermixCapabilities.length > 0 ? (
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {a.hermixCapabilities.map((c) => (
                    <span
                      key={c}
                      className="rounded-sm bg-accent px-1.5 py-0.5 text-xs text-accent-foreground"
                    >
                      {c}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onRegenerate(a.id)}
            >
              重置 Token
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
