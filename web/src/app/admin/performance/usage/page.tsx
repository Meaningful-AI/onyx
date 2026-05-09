"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";
import { Table, Text, createTableColumns } from "@opal/components";
import { IllustrationContent } from "@opal/layouts";
import SvgNoResult from "@opal/illustrations/no-result";
import { AdminDateRangeSelector } from "@/components/dateRangeSelectors/AdminDateRangeSelector";
import type { DateRange } from "@/components/dateRangeSelectors/AdminDateRangeSelector";
import {
  convertDateToEndOfDay,
  convertDateToStartOfDay,
  getXDaysAgo,
} from "@/components/dateRangeSelectors/dateUtils";
import { AreaChartDisplay } from "@/components/ui/areaChart";
import * as SettingsLayouts from "@/layouts/settings-layouts";
import { ADMIN_ROUTES } from "@/lib/admin-routes";
import { errorHandlingFetcher } from "@/lib/fetcher";
import { buildApiPath } from "@/lib/urlBuilder";
import SimpleLoader from "@/refresh-components/loaders/SimpleLoader";
import Message from "@/refresh-components/messages/Message";

interface DailyUsageStat {
  date: string;
  chat_queries: number;
  search_queries: number;
  total_queries: number;
  active_users: number;
}

interface TopUserUsageStat {
  user_email: string;
  chat_queries: number;
  search_queries: number;
  total_queries: number;
}

interface TopAssistantUsageStat {
  assistant_id: number | null;
  assistant_name: string;
  messages: number;
  unique_users: number;
}

interface AdminUsageStatsSnapshot {
  start: string;
  end: string;
  chat_queries: number;
  search_queries: number;
  total_queries: number;
  active_users: number;
  chat_sessions: number;
  positive_feedback: number;
  negative_feedback: number;
  daily: DailyUsageStat[];
  top_users: TopUserUsageStat[];
  top_assistants: TopAssistantUsageStat[];
}

interface MetricCardProps {
  label: string;
  value: number;
  detail: string;
}

interface ChartRow {
  Date: string;
  "Chat queries": number;
  "Search queries": number;
  "Active users": number;
}

const route = ADMIN_ROUTES.USAGE;
const topUsersTc = createTableColumns<TopUserUsageStat>();
const topAssistantsTc = createTableColumns<TopAssistantUsageStat>();

function formatInteger(value: number) {
  return new Intl.NumberFormat().format(value);
}

function getDateKey(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatDateLabel(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(new Date(`${value}T00:00:00`));
}

function buildDailyChartData(
  daily: DailyUsageStat[],
  timeRange: DateRange
): ChartRow[] {
  if (!timeRange?.from || !timeRange.to) {
    return [];
  }

  const byDate = new Map(daily.map((row) => [row.date, row]));
  const rows: ChartRow[] = [];
  const current = new Date(timeRange.from);
  current.setHours(0, 0, 0, 0);
  const end = new Date(timeRange.to);
  end.setHours(0, 0, 0, 0);

  while (current <= end) {
    const key = getDateKey(current);
    const row = byDate.get(key);
    rows.push({
      Date: key,
      "Chat queries": row?.chat_queries ?? 0,
      "Search queries": row?.search_queries ?? 0,
      "Active users": row?.active_users ?? 0,
    });
    current.setDate(current.getDate() + 1);
  }

  return rows;
}

function MetricCard({ label, value, detail }: MetricCardProps) {
  return (
    <div className="rounded-08 border bg-background-neutral-00 p-4">
      <p className="text-xs font-medium uppercase text-text-02">{label}</p>
      <p className="mt-2 text-3xl font-semibold text-text-05">
        {formatInteger(value)}
      </p>
      <p className="mt-1 text-sm text-text-03">{detail}</p>
    </div>
  );
}

const topUserColumns = [
  topUsersTc.column("user_email", {
    header: "User",
    weight: 40,
    cell: (value) => <Text font="secondary-mono">{value}</Text>,
  }),
  topUsersTc.column("total_queries", {
    header: "Queries",
    weight: 18,
    cell: (value) => formatInteger(value),
  }),
  topUsersTc.column("chat_queries", {
    header: "Chat",
    weight: 16,
    cell: (value) => formatInteger(value),
  }),
  topUsersTc.column("search_queries", {
    header: "Search",
    weight: 16,
    cell: (value) => formatInteger(value),
  }),
];

const topAssistantColumns = [
  topAssistantsTc.column("assistant_name", {
    header: "Assistant",
    weight: 44,
    cell: (value) => (
      <span className="text-sm leading-5 text-text-04">{value}</span>
    ),
  }),
  topAssistantsTc.column("messages", {
    header: "Messages",
    weight: 18,
    cell: (value) => formatInteger(value),
  }),
  topAssistantsTc.column("unique_users", {
    header: "Users",
    weight: 18,
    cell: (value) => formatInteger(value),
  }),
];

export default function UsageStatisticsPage() {
  const [timeRange, setTimeRange] = useState<DateRange>({
    from: getXDaysAgo(30),
    to: new Date(),
  });

  const url = useMemo(() => {
    if (!timeRange?.from || !timeRange.to) {
      return null;
    }

    return buildApiPath("/api/admin/usage-stats", {
      start: convertDateToStartOfDay(timeRange.from)?.toISOString(),
      end: convertDateToEndOfDay(timeRange.to)?.toISOString(),
      limit: 10,
    });
  }, [timeRange]);

  const { data, error, isLoading } = useSWR<AdminUsageStatsSnapshot>(
    url,
    errorHandlingFetcher
  );

  const chartData = useMemo(
    () => buildDailyChartData(data?.daily ?? [], timeRange),
    [data?.daily, timeRange]
  );

  return (
    <SettingsLayouts.Root width="full">
      <SettingsLayouts.Header icon={route.icon} title={route.title} separator>
        <AdminDateRangeSelector
          value={timeRange}
          onValueChange={setTimeRange}
        />
      </SettingsLayouts.Header>

      <SettingsLayouts.Body>
        {error ? (
          <Message
            static
            error
            text="Unable to load usage statistics"
            description={error.message}
            close={false}
            className="w-full"
          />
        ) : isLoading && !data ? (
          <div className="flex justify-center py-12">
            <SimpleLoader />
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <MetricCard
                label="Total queries"
                value={data?.total_queries ?? 0}
                detail={`${formatInteger(
                  data?.chat_queries ?? 0
                )} chat / ${formatInteger(data?.search_queries ?? 0)} search`}
              />
              <MetricCard
                label="Active users"
                value={data?.active_users ?? 0}
                detail="Users with chat or search activity"
              />
              <MetricCard
                label="Chat sessions"
                value={data?.chat_sessions ?? 0}
                detail="Sessions created in range"
              />
              <MetricCard
                label="Feedback"
                value={
                  (data?.positive_feedback ?? 0) +
                  (data?.negative_feedback ?? 0)
                }
                detail={`${formatInteger(
                  data?.positive_feedback ?? 0
                )} positive / ${formatInteger(
                  data?.negative_feedback ?? 0
                )} negative`}
              />
            </div>

            <AreaChartDisplay
              data={chartData}
              categories={["Chat queries", "Search queries", "Active users"]}
              index="Date"
              colors={["#4f46e5", "#059669", "#c026d3"]}
              allowDecimals={false}
              title="Daily activity"
              description="Query volume and active users"
              xAxisFormatter={formatDateLabel}
              yAxisFormatter={(value) => formatInteger(value)}
            />

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
              <section className="flex flex-col gap-3">
                <h2 className="text-base font-semibold text-text-05">
                  Top users
                </h2>
                <Table
                  data={data?.top_users ?? []}
                  columns={topUserColumns}
                  getRowId={(row) => row.user_email}
                  pageSize={10}
                  emptyState={
                    <IllustrationContent
                      illustration={SvgNoResult}
                      title="No users found"
                      description="No user activity was found for this range."
                    />
                  }
                  footer={{ units: "users" }}
                />
              </section>

              <section className="flex flex-col gap-3">
                <h2 className="text-base font-semibold text-text-05">
                  Top assistants
                </h2>
                <Table
                  data={data?.top_assistants ?? []}
                  columns={topAssistantColumns}
                  getRowId={(row) =>
                    row.assistant_id === null
                      ? "no-assistant"
                      : String(row.assistant_id)
                  }
                  pageSize={10}
                  emptyState={
                    <IllustrationContent
                      illustration={SvgNoResult}
                      title="No assistants found"
                      description="No assistant activity was found for this range."
                    />
                  }
                  footer={{ units: "assistants" }}
                />
              </section>
            </div>
          </div>
        )}
      </SettingsLayouts.Body>
    </SettingsLayouts.Root>
  );
}
