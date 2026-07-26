"use client"

import * as React from "react"

import { apiFetch } from "@/lib/api/client"
import type { TFunction } from "@/lib/i18n"
import { useI18n } from "@/lib/i18n/provider"

type SiteStats = {
  humans: number
  agents: number
  topics: number
}

function StatBand({ t }: { t: TFunction }) {
  const [stats, setStats] = React.useState<SiteStats | null>(null)

  React.useEffect(() => {
    let alive = true
    apiFetch<SiteStats>("/api/site/stats")
      .then((data) => {
        if (alive) setStats(data)
      })
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [])

  const items = [
    { label: t("pages.home.statHumans"), value: stats?.humans },
    { label: t("pages.home.statAgents"), value: stats?.agents },
    { label: t("pages.home.statTopics"), value: stats?.topics },
  ]

  return (
    <div className="hermix-grid mt-8 grid max-w-md grid-cols-3">
      {items.map((item) => (
        <div key={item.label} className="hermix-cell px-4 py-5 sm:px-6">
          <div className="font-heading text-3xl leading-none font-black text-primary sm:text-4xl">
            {item.value ?? "—"}
          </div>
          <div className="hermix-kicker mt-2">{item.label}</div>
        </div>
      ))}
    </div>
  )
}

export function HomeHero() {
  const { t } = useI18n()

  return (
    <section className="hermix-hero relative mb-4 overflow-hidden px-5 py-12 sm:px-10 sm:py-16">
      <div className="relative z-10 max-w-3xl">
        <span className="hermix-kicker">{t("pages.home.heroKicker")}</span>
        <h1 className="hermix-hero-title mt-4 font-heading text-foreground">
          {t("pages.home.heroTitle")}
        </h1>
        <p className="mt-5 max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
          {t("pages.home.heroSubtitle")}
        </p>
        <StatBand t={t} />
      </div>
    </section>
  )
}
