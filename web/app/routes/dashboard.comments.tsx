"use client"

import {
  DashboardDataPage,
  type DashboardDataPageConfig,
} from "@/components/dashboard/data"
import * as dashboardData from "@/components/dashboard/data/dashboard-data-route-utils"
import { useI18n } from "@/lib/i18n/provider"
import { PERMISSIONS } from "@/lib/auth/permissions.generated"

export default function DashboardCommentsRoute() {
  const { t } = useI18n()
  const config: DashboardDataPageConfig = {
    title: dashboardData.title(t, "comments"),
    description: dashboardData.desc(t, "comments"),
    listEndpoint: "/api/admin/comment/list",
    viewPermission: PERMISSIONS.DASHBOARD_COMMENT_VIEW,
    filters: [
      { name: "id", label: dashboardData.label(t, "id") },
      { name: "entityId", label: dashboardData.label(t, "entityId") },
      { name: "content", label: dashboardData.label(t, "content") },
      {
        name: "entityType",
        label: dashboardData.label(t, "entityType"),
        type: "select",
        options: [
          { label: t("dashboard.commentEntityTypes.topic"), value: "topic" },
          {
            label: t("dashboard.commentEntityTypes.comment"),
            value: "comment",
          },
        ],
      },
      {
        name: "status",
        label: dashboardData.label(t, "status"),
        type: "select",
        options: [
          { label: t("dashboard.topicFeed.statusNormal"), value: 0 },
          { label: t("dashboard.topicFeed.statusDeleted"), value: 1 },
          { label: t("dashboard.topicFeed.statusReview"), value: 2 },
        ],
      },
    ],
    columns: [
      { key: "id", label: dashboardData.label(t, "id") },
      {
        key: "user",
        label: dashboardData.label(t, "user"),
        render: (record) => {
          const user = record.user as Record<string, unknown> | undefined
          return user ? String(user.nickname || user.username || "-") : "-"
        },
      },
      {
        key: "content",
        label: dashboardData.label(t, "content"),
        className: "min-w-72",
      },
      {
        key: "entityId",
        label: dashboardData.label(t, "entityId"),
        render: (record) =>
          record.entityUrl ? (
            <a
              href={String(record.entityUrl)}
              target="_blank"
              rel="noreferrer"
              className="hover:underline"
            >
              {String(record.entityId ?? "-")}
            </a>
          ) : (
            String(record.entityId ?? "-")
          ),
      },
      {
        key: "status",
        label: dashboardData.label(t, "status"),
        render: (record) => commentStatusText(t, record.status),
      },
      {
        key: "createTime",
        label: dashboardData.label(t, "createTime"),
        render: (record) => dashboardData.dateCell(record.createTime),
      },
    ],
    rowActions: [
      {
        label: t("dashboard.actions.audit"),
        endpoint: "/api/admin/comment/audit",
        permission: PERMISSIONS.DASHBOARD_COMMENT_AUDIT,
        payload: (record) => ({ id: record.id as number }),
        visible: (record) => Number(record.status ?? 0) === 2,
        successMessage: t("dashboard.messages.audited"),
      },
      {
        label: t("dashboard.actions.delete"),
        endpoint: "/api/admin/comment/delete",
        permission: PERMISSIONS.DASHBOARD_COMMENT_DELETE,
        payload: (record) => ({ id: record.id as number }),
        visible: (record) => Number(record.status ?? 0) !== 1,
        confirm: t("dashboard.confirmDelete"),
        successMessage: t("dashboard.messages.deleted"),
      },
    ],
  }

  return <DashboardDataPage config={config} />
}

function commentStatusText(
  t: ReturnType<typeof useI18n>["t"],
  status: unknown
) {
  switch (Number(status ?? 0)) {
    case 1:
      return t("dashboard.topicFeed.statusDeleted")
    case 2:
      return t("dashboard.topicFeed.statusReview")
    default:
      return t("dashboard.topicFeed.statusNormal")
  }
}
