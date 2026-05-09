"use client";

import { useCallback, useMemo, useState } from "react";
import useSWR from "swr";
import { Button, Table, Text, createTableColumns } from "@opal/components";
import { IllustrationContent } from "@opal/layouts";
import SvgNoResult from "@opal/illustrations/no-result";
import {
  SvgDownload,
  SvgHistory,
  SvgSearch,
  SvgTextLines,
  SvgUser,
} from "@opal/icons";
import * as SettingsLayouts from "@/layouts/settings-layouts";
import { ADMIN_ROUTES } from "@/lib/admin-routes";
import { errorHandlingFetcher } from "@/lib/fetcher";
import { cn } from "@/lib/utils";
import InputSelect from "@/refresh-components/inputs/InputSelect";
import InputTypeIn from "@/refresh-components/inputs/InputTypeIn";
import SimpleLoader from "@/refresh-components/loaders/SimpleLoader";
import Message from "@/refresh-components/messages/Message";
import { toast } from "@/hooks/useToast";

type QueryLogSource = "all" | "chat" | "search";

interface QueryLogEntry {
  id: string;
  source: "chat" | "search";
  user_email: string;
  query: string;
  time_created: string;
  chat_session_id: string | null;
  chat_session_name: string | null;
  assistant_name: string | null;
}

interface QueryLogResponse {
  items: QueryLogEntry[];
  total_items: number;
}

interface QueryLogUser {
  email: string;
  display_email: string;
}

const route = ADMIN_ROUTES.QUERY_HISTORY;
const PAGE_SIZE = 50;
const ALL_USERS_VALUE = "__all_users__";
const tc = createTableColumns<QueryLogEntry>();

function toStartIso(date: string) {
  if (!date) return undefined;
  return new Date(`${date}T00:00:00`).toISOString();
}

function toEndIso(date: string) {
  if (!date) return undefined;
  return new Date(`${date}T23:59:59.999`).toISOString();
}

function buildQueryLogParams({
  page,
  source,
  search,
  userEmail,
  startDate,
  endDate,
  includePage,
}: {
  page: number;
  source: QueryLogSource;
  search: string;
  userEmail: string;
  startDate: string;
  endDate: string;
  includePage: boolean;
}) {
  const params = new URLSearchParams();
  if (includePage) {
    params.set("page_num", String(page));
    params.set("page_size", String(PAGE_SIZE));
  }
  if (source !== "all") params.set("source", source);
  if (search.trim()) params.set("q", search.trim());
  if (userEmail.trim()) params.set("user_email", userEmail.trim());

  const startIso = toStartIso(startDate);
  const endIso = toEndIso(endDate);
  if (startIso) params.set("start_time", startIso);
  if (endIso) params.set("end_time", endIso);

  return params;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

async function downloadQueryLog(params: URLSearchParams) {
  const response = await fetch(`/api/admin/query-log/download?${params}`);
  if (!response.ok) {
    throw new Error("Failed to download query log.");
  }

  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = "query-log.csv";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export default function QueryHistoryPage() {
  const [page, setPage] = useState(0);
  const [source, setSource] = useState<QueryLogSource>("all");
  const [search, setSearch] = useState("");
  const [userEmail, setUserEmail] = useState(ALL_USERS_VALUE);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [expandedQueryIds, setExpandedQueryIds] = useState<Set<string>>(
    () => new Set()
  );

  const queryParams = useMemo(
    () =>
      buildQueryLogParams({
        page,
        source,
        search,
        userEmail: userEmail === ALL_USERS_VALUE ? "" : userEmail,
        startDate,
        endDate,
        includePage: true,
      }),
    [endDate, page, search, source, startDate, userEmail]
  );

  const downloadParams = useMemo(
    () =>
      buildQueryLogParams({
        page,
        source,
        search,
        userEmail: userEmail === ALL_USERS_VALUE ? "" : userEmail,
        startDate,
        endDate,
        includePage: false,
      }),
    [endDate, page, search, source, startDate, userEmail]
  );

  const { data, error, isLoading } = useSWR<QueryLogResponse>(
    `/api/admin/query-log?${queryParams}`,
    errorHandlingFetcher
  );

  const { data: users } = useSWR<QueryLogUser[]>(
    "/api/admin/query-log/users",
    errorHandlingFetcher
  );

  const toggleExpandedQuery = useCallback((id: string) => {
    setExpandedQueryIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const columns = useMemo(
    () => [
      tc.qualifier({
        content: "icon",
        getContent: (row) => (row.source === "chat" ? SvgTextLines : SvgSearch),
        background: true,
      }),
      tc.column("query", {
        header: "Query",
        weight: 42,
        cell: (value, row) => {
          const isExpanded = expandedQueryIds.has(row.id);
          const description =
            row.chat_session_name || row.assistant_name || row.source;

          return (
            <button
              type="button"
              className="group flex w-full flex-col items-start gap-1 text-left"
              aria-expanded={isExpanded}
              onClick={() => toggleExpandedQuery(row.id)}
            >
              <span
                className={cn(
                  "whitespace-pre-wrap break-words text-sm leading-5 text-text-04",
                  !isExpanded && "line-clamp-2"
                )}
              >
                {value}
              </span>
              <span className="line-clamp-1 text-xs text-text-02">
                {description}
              </span>
            </button>
          );
        },
      }),
      tc.column("user_email", {
        header: "User",
        weight: 20,
        cell: (value) => <Text font="secondary-mono">{value}</Text>,
      }),
      tc.column("source", {
        header: "Source",
        weight: 10,
        cell: (value) => (value === "chat" ? "Chat" : "Search"),
      }),
      tc.column("time_created", {
        header: "Date",
        weight: 18,
        cell: (value) => formatDate(value),
      }),
    ],
    [expandedQueryIds, toggleExpandedQuery]
  );

  const resetPage = () => setPage(0);

  return (
    <SettingsLayouts.Root width="full">
      <SettingsLayouts.Header
        icon={route.icon}
        title={route.title}
        description="Admin query log"
        rightChildren={
          <Button
            icon={SvgDownload}
            onClick={() => {
              downloadQueryLog(downloadParams).catch((err) => {
                toast.error(
                  err instanceof Error
                    ? err.message
                    : "Failed to download query log."
                );
              });
            }}
          >
            Download CSV
          </Button>
        }
        separator
      >
        <div className="grid grid-cols-1 gap-2 md:grid-cols-[minmax(220px,2fr)_minmax(180px,1fr)_160px_160px_160px]">
          <InputTypeIn
            leftSearchIcon
            value={search}
            placeholder="Search queries"
            onChange={(event) => {
              setSearch(event.target.value);
              resetPage();
            }}
          />
          <InputSelect
            value={userEmail}
            onValueChange={(value) => {
              setUserEmail(value);
              resetPage();
            }}
          >
            <InputSelect.Trigger placeholder="Filter by user" />
            <InputSelect.Content>
              <InputSelect.Item value={ALL_USERS_VALUE} icon={SvgUser}>
                All users
              </InputSelect.Item>
              {users?.map((user) => (
                <InputSelect.Item
                  key={user.email}
                  value={user.email}
                  icon={SvgUser}
                >
                  {user.display_email}
                </InputSelect.Item>
              ))}
            </InputSelect.Content>
          </InputSelect>
          <InputSelect
            value={source}
            onValueChange={(value) => {
              setSource(value as QueryLogSource);
              resetPage();
            }}
          >
            <InputSelect.Trigger />
            <InputSelect.Content>
              <InputSelect.Item value="all" icon={SvgHistory}>
                All sources
              </InputSelect.Item>
              <InputSelect.Item value="chat" icon={SvgTextLines}>
                Chat
              </InputSelect.Item>
              <InputSelect.Item value="search" icon={SvgSearch}>
                Search
              </InputSelect.Item>
            </InputSelect.Content>
          </InputSelect>
          <InputTypeIn
            type="date"
            value={startDate}
            onChange={(event) => {
              setStartDate(event.target.value);
              resetPage();
            }}
          />
          <InputTypeIn
            type="date"
            value={endDate}
            onChange={(event) => {
              setEndDate(event.target.value);
              resetPage();
            }}
          />
        </div>
      </SettingsLayouts.Header>

      <SettingsLayouts.Body>
        {error ? (
          <Message
            static
            error
            text="Unable to load query log"
            description={error.message}
            close={false}
            className="w-full"
          />
        ) : isLoading && !data ? (
          <div className="flex justify-center py-12">
            <SimpleLoader />
          </div>
        ) : (
          <Table
            data={data?.items ?? []}
            columns={columns}
            getRowId={(row) => row.id}
            pageSize={PAGE_SIZE}
            serverSide={{
              totalItems: data?.total_items ?? 0,
              isLoading,
              onPaginationChange: (nextPage) => setPage(nextPage),
              onSortingChange: () => undefined,
              onSearchTermChange: () => undefined,
            }}
            emptyState={
              <IllustrationContent
                illustration={SvgNoResult}
                title="No queries found"
                description="No query log entries match the current filters."
              />
            }
            footer={{ units: "queries" }}
          />
        )}
      </SettingsLayouts.Body>
    </SettingsLayouts.Root>
  );
}
