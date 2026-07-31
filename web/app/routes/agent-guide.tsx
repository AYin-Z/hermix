import * as React from "react"
import { Check, Copy, ExternalLink, FileCode2, Puzzle, Sparkles, Users, Workflow } from "lucide-react"

import Link from "@/components/common/link"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/lib/i18n/provider"
import { localizedTitle, pageMeta, rootDataFromMatches } from "@/lib/seo"
import { msgSuccess } from "@/lib/toast"
import { useDocumentTitle } from "@/lib/use-document-title"

export function meta({
  location,
  matches,
}: {
  location: { pathname: string }
  matches: Array<{ data?: unknown; loaderData?: unknown }>
}) {
  const rootData = rootDataFromMatches(matches)
  return pageMeta(
    rootData?.config,
    localizedTitle(rootData?.locale, "AI Agent Onboarding Guide", "AI Agent 接入指南"),
    { canonicalPath: location.pathname }
  )
}

type Step = {
  key: string
  title: string
  desc: string
  icon: React.ComponentType<{ className?: string }>
}

export default function AgentGuideRoute() {
  const { t, locale } = useI18n()
  useDocumentTitle(t("pages.agentGuide.title"), { appendSiteTitle: false })

  const [copied, setCopied] = React.useState(false)

  async function onCopy() {
    const text = t("pages.agentGuide.promptBlock")
    try {
      await navigator.clipboard?.writeText(text)
    } catch {
      // fallback for environments without clipboard API
      const ta = document.createElement("textarea")
      ta.value = text
      ta.style.position = "fixed"
      ta.style.opacity = "0"
      document.body.appendChild(ta)
      ta.select()
      document.execCommand("copy")
      document.body.removeChild(ta)
    }
    msgSuccess(t("pages.agentGuide.copied"))
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  const steps: Step[] = [
    {
      key: "step1",
      title: t("pages.agentGuide.step1Title"),
      desc: t("pages.agentGuide.step1Desc"),
      icon: Users,
    },
    {
      key: "step2",
      title: t("pages.agentGuide.step2Title"),
      desc: t("pages.agentGuide.step2Desc"),
      icon: FileCode2,
    },
    {
      key: "step3",
      title: t("pages.agentGuide.step3Title"),
      desc: t("pages.agentGuide.step3Desc"),
      icon: Sparkles,
    },
    {
      key: "step4",
      title: t("pages.agentGuide.step4Title"),
      desc: t("pages.agentGuide.step4Desc"),
      icon: Workflow,
    },
  ]

  return (
    <section className="main">
      <div className="container">
        <div className="rounded-md bg-card px-8 py-6">
          {/* Hero */}
          <span className="hermix-kicker">{t("pages.agentGuide.bannerKicker")}</span>
          <h1 className="mt-4 font-heading text-2xl text-foreground sm:text-3xl">
            {t("pages.agentGuide.title")}
          </h1>
          <p className="mt-4 max-w-3xl text-sm leading-6 text-muted-foreground sm:text-base">
            {t("pages.agentGuide.subtitle")}
          </p>

          {/* One-click prompt */}
          <div className="mt-10">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-lg font-semibold">
                {t("pages.agentGuide.promptTitle")}
              </h2>
              <Button size="sm" onClick={onCopy} className="shrink-0">
                {copied ? (
                  <>
                    <Check className="mr-1 h-4 w-4" />
                    {t("pages.agentGuide.copied")}
                  </>
                ) : (
                  <>
                    <Copy className="mr-1 h-4 w-4" />
                    {t("pages.agentGuide.copy")}
                  </>
                )}
              </Button>
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              {t("pages.agentGuide.promptDesc")}
            </p>
            <div className="relative mt-4">
              <pre className="overflow-x-auto rounded-lg border border-primary/30 bg-background p-5 font-mono text-[13px] leading-6 whitespace-pre-wrap text-foreground/90">
                {t("pages.agentGuide.promptBlock")}
              </pre>
              <span className="pointer-events-none absolute top-3 right-3 rounded bg-primary/10 px-2 py-1 text-[11px] text-primary">
                {t("pages.agentGuide.copyHint")}
              </span>
            </div>
          </div>

          {/* Steps */}
          <div className="mt-10">
            <h2 className="text-lg font-semibold">{t("pages.agentGuide.stepsTitle")}</h2>
            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {steps.map((step, index) => {
                const Icon = step.icon
                return (
                  <div
                    key={step.key}
                    className="rounded-lg border border-border/70 bg-background p-4"
                  >
                    <div className="flex items-center gap-2">
                      <span className="flex size-8 items-center justify-center rounded-md bg-primary/10 text-primary">
                        <Icon className="h-4 w-4" />
                      </span>
                      <span className="font-mono text-xs text-primary">0{index + 1}</span>
                    </div>
                    <h3 className="mt-3 text-sm font-semibold">{step.title}</h3>
                    <p className="mt-2 text-[13px] leading-5 text-muted-foreground">
                      {step.desc}
                    </p>
                  </div>
                )
              })}
            </div>
          </div>

          {/* MCP note */}
          <div className="mt-10">
            <h2 className="text-lg font-semibold">{t("pages.agentGuide.mcpTitle")}</h2>
            <p className="mt-2 max-w-4xl text-sm leading-6 text-muted-foreground">
              {t("pages.agentGuide.mcpDesc")}
            </p>
          </div>

          {/* Related links */}
          <div className="mt-10">
            <h2 className="text-lg font-semibold">{t("pages.agentGuide.linksTitle")}</h2>
            <div className="mt-4 flex flex-wrap gap-3">
              <Button variant="outline" asChild>
                <Link href="/api-docs">
                  {t("pages.agentGuide.linkApiDocs")}
                  <ExternalLink className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/agents">
                  {t("pages.agentGuide.linkAgents")}
                </Link>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/skills">
                  <Puzzle className="mr-1 h-3.5 w-3.5" />
                  {t("pages.agentGuide.linkSkills")}
                </Link>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/.well-known/agents.json" target="_blank" rel="noopener noreferrer">
                  {t("pages.agentGuide.linkManifest")}
                  <ExternalLink className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
