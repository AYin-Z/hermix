"use client"

import * as React from "react"
import { ArrowRight, Bot, Check, Copy } from "lucide-react"

import Link from "@/components/common/link"
import { Button } from "@/components/ui/button"
import { useI18n } from "@/lib/i18n/provider"
import { msgSuccess } from "@/lib/toast"

export function AgentGuideBanner() {
  const { t } = useI18n()
  const [copied, setCopied] = React.useState(false)

  async function onCopy() {
    const text = t("pages.agentGuide.promptBlock")
    try {
      await navigator.clipboard?.writeText(text)
    } catch {
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

  return (
    <div className="relative mt-4 overflow-hidden rounded-xl border border-primary/40 bg-gradient-to-br from-primary/15 via-background to-background px-5 py-6 sm:px-8">
      <div className="flex flex-col items-start justify-between gap-4 lg:flex-row lg:items-center">
        <div className="flex items-start gap-4">
          <span className="flex size-12 shrink-0 items-center justify-center rounded-lg bg-primary/15 text-primary">
            <Bot className="h-6 w-6" />
          </span>
          <div>
            <span className="hermix-kicker">{t("pages.agentGuide.bannerKicker")}</span>
            <h2 className="mt-1 font-heading text-lg text-foreground">
              {t("pages.agentGuide.bannerTitle")}
            </h2>
            <p className="mt-1 max-w-xl text-sm leading-5 text-muted-foreground">
              {t("pages.agentGuide.bannerDesc")}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            onClick={onCopy}
            variant="ghost"
            className="border border-primary/60 bg-primary/10 text-primary hover:bg-primary/20 hover:text-primary"
          >
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
          <Button asChild>
            <Link href="/agent-guide">
              {t("pages.agentGuide.bannerCta")}
              <ArrowRight className="ml-1 h-4 w-4" />
            </Link>
          </Button>
        </div>
      </div>
    </div>
  )
}
