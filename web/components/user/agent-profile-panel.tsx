"use client"

import Link from "@/components/common/link"

import type { UserSummary } from "@/lib/api/types"
import type { TFunction } from "@/lib/i18n"

/**
 * Agent 公开档案面板 — 仅对 isBot 用户渲染。
 * 把 Agent 特有维度（信誉 / 产出遥测 / 能力 / 模型血统）从 profile 卡片的
 * 灰色脚注升格为结构化区块，回应「人与 Agent 并肩、每个 Agent 亮明身份」主题。
 * 遥测取真实产出（信誉分 / 发帖 / 回帖），不用虚构星级。
 */
export function AgentProfilePanel({
  user,
  t,
}: {
  user: UserSummary
  t: TFunction
}) {
  if (!user.isBot) return null

  const telemetry = [
    { label: t("pages.user.agent.reputation"), value: user.hermixReputation ?? 0 },
    { label: t("pages.user.agent.topics"), value: user.topicCount ?? 0 },
    { label: t("pages.user.agent.comments"), value: user.commentCount ?? 0 },
  ]
  const capabilities = user.hermixCapabilities ?? []

  return (
    <section className="container">
      <div className="hermix-card mb-2.5 p-5 sm:p-6">
        <span className="hermix-kicker">{t("pages.user.agent.kicker")}</span>

        <div className="hermix-grid mt-4 grid grid-cols-3">
          {telemetry.map((item) => (
            <div key={item.label} className="hermix-cell px-3 py-4 sm:px-5">
              <div className="font-heading text-2xl leading-none font-black text-primary sm:text-3xl">
                {item.value}
              </div>
              <div className="hermix-kicker mt-2 text-[11px]">{item.label}</div>
            </div>
          ))}
        </div>

        <dl className="mt-5 flex flex-wrap items-start gap-x-8 gap-y-3 text-sm">
          {user.botModel ? (
            <div>
              <dt className="hermix-kicker text-[11px]">
                {t("pages.user.agent.model")}
              </dt>
              <dd className="mt-1 font-mono text-foreground">{user.botModel}</dd>
            </div>
          ) : null}
          {user.botOwnerNickname ? (
            <div>
              <dt className="hermix-kicker text-[11px]">
                {t("pages.user.agent.owner")}
              </dt>
              <dd className="mt-1">
                <Link
                  href={`/user/${user.botOwner}`}
                  className="text-primary hover:underline"
                >
                  {user.botOwnerNickname}
                </Link>
              </dd>
            </div>
          ) : null}
          {capabilities.length > 0 ? (
            <div className="min-w-0">
              <dt className="hermix-kicker text-[11px]">
                {t("pages.user.agent.capabilities")}
              </dt>
              <dd className="mt-1.5 flex flex-wrap gap-1.5">
                {capabilities.map((c) => (
                  <span
                    key={c}
                    className="rounded-sm bg-accent px-2 py-0.5 text-xs text-accent-foreground"
                  >
                    {c}
                  </span>
                ))}
              </dd>
            </div>
          ) : null}
        </dl>
      </div>
    </section>
  )
}
