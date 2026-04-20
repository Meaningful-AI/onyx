"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFormikContext } from "formik";
import { Text } from "@opal/components";
import AppInputBar, {
  AppInputBarHandle,
} from "@/sections/input/AppInputBar";
import ModelSelector from "@/refresh-components/popovers/ModelSelector";
import { STARTER_MESSAGES_EXAMPLES, DEFAULT_CONTEXT_TOKENS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useFilters, useLlmManager } from "@/lib/hooks";
import useMultiModelChat from "@/hooks/useMultiModelChat";
import { MinimalPersonaSnapshot } from "@/app/admin/agents/interfaces";
import type { ChatState } from "@/app/app/interfaces";

interface ChatMessage {
  role: "assistant" | "user";
  content: string;
}

const WELCOME_MESSAGE: ChatMessage = {
  role: "assistant",
  content:
    "Tell me about the agent you want to create. What should it do? Who is it for?",
};

const WIZARD_KEYS = [
  "name",
  "description",
  "instructions",
  "starter_messages",
  "web_search",
  "image_generation",
  "code_interpreter",
] as const;

interface WizardCurrentValues {
  name: string;
  description: string;
  instructions: string;
  starter_messages: string[];
  web_search: boolean;
  image_generation: boolean;
  code_interpreter: boolean;
}

/** Minimal persona so AppInputBar hides Actions + Deep Research (no search tools). */
const WIZARD_INPUT_BAR_AGENT: MinimalPersonaSnapshot = {
  id: -1,
  name: "Agent wizard",
  description: "",
  tools: [],
  starter_messages: null,
  document_sets: [],
  is_public: false,
  is_listed: false,
  display_priority: null,
  is_featured: false,
  builtin_persona: false,
  owner: null,
};

const AVAILABLE_CONTEXT_TOKENS = Number(DEFAULT_CONTEXT_TOKENS) * 0.5;

/** Thread width; narrow wizard column must not overflow (no extra horizontal margins on w-full). */
const THREAD_BUBBLE =
  "max-w-[min(100%,720px)] w-full min-w-0 shrink";

interface AgentBuilderChatProps {
  markFieldUpdated: (field: string) => void;
}

function pickWizardValues(values: Record<string, unknown>): WizardCurrentValues {
  return {
    name: String(values.name ?? ""),
    description: String(values.description ?? ""),
    instructions: String(values.instructions ?? ""),
    starter_messages: Array.isArray(values.starter_messages)
      ? (values.starter_messages as string[]).map((s) => String(s))
      : [],
    web_search: Boolean(values.web_search),
    image_generation: Boolean(values.image_generation),
    code_interpreter: Boolean(values.code_interpreter),
  };
}

function normalizeStarters(raw: unknown): string[] {
  const arr = Array.isArray(raw) ? raw.map((x) => String(x)) : [];
  return Array.from(
    { length: STARTER_MESSAGES_EXAMPLES.length },
    (_, i) => arr[i] ?? ""
  );
}

export default function AgentBuilderChat({
  markFieldUpdated,
}: AgentBuilderChatProps) {
  const { values, setFieldValue, setFieldTouched } =
    useFormikContext<Record<string, unknown>>();
  const [messages, setMessages] = useState<ChatMessage[]>([WELCOME_MESSAGE]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [noLlmError, setNoLlmError] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const chatInputBarRef = useRef<AppInputBarHandle | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const filterManager = useFilters();
  // No liveAgent: use global provider list. A synthetic id (e.g. -1) would hit
  // GET /api/llm/persona/-1/providers and 404. AppInputBar still gets tools: [] via selectedAgent.
  const llmManager = useLlmManager(undefined, undefined);
  const multiModel = useMultiModelChat(llmManager);

  const currentValues = useMemo(
    () => pickWizardValues(values),
    [values]
  );

  const chatState: ChatState = isStreaming ? "streaming" : "input";

  const stopGenerating = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setIsStreaming(false);
  }, []);

  const streamResponse = useCallback(
    async (nextMessages: ChatMessage[]) => {
      setIsStreaming(true);
      setNoLlmError(false);
      abortRef.current?.abort();
      abortRef.current = new AbortController();
      const { signal } = abortRef.current;

      let res: Response;
      try {
        res = await fetch("/api/agent-wizard-chat", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            messages: nextMessages.map(({ role, content }) => ({
              role,
              content,
            })),
            currentValues,
          }),
          signal,
        });
      } catch (e) {
        setIsStreaming(false);
        const aborted =
          (e instanceof DOMException && e.name === "AbortError") ||
          (e instanceof Error && e.name === "AbortError");
        if (aborted) {
          return;
        }
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant",
            content: "Could not reach the assistant. Try again in a moment.",
          },
        ]);
        return;
      }

      if (res.status === 503) {
        setIsStreaming(false);
        setNoLlmError(true);
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant",
            content:
              "No LLM is configured for this workspace. You can still fill out the form on the right.",
          },
        ]);
        return;
      }

      if (!res.ok || !res.body) {
        setIsStreaming(false);
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant",
            content: "Something went wrong. Please try again.",
          },
        ]);
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let fullResponse = "";

      setMessages((prev) => [...prev, { role: "assistant", content: "" }]);

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const parts = buffer.split("\n\n");
          buffer = parts.pop() ?? "";

          for (const part of parts) {
            const line = part.trim();
            if (!line.startsWith("data:")) continue;
            const payload = line.slice(5).trim();
            try {
              const parsed = JSON.parse(payload) as {
                delta?: string;
                error?: string;
              };
              if (parsed.error) {
                continue;
              }
              if (parsed.delta) {
                fullResponse += parsed.delta;
                setMessages((prev) => {
                  const copy = [...prev];
                  const last = copy[copy.length - 1];
                  if (last && last.role === "assistant") {
                    copy[copy.length - 1] = {
                      ...last,
                      content: last.content + parsed.delta,
                    };
                  }
                  return copy;
                });
              }
            } catch {
              /* ignore malformed chunk */
            }
          }
        }
      } catch (e) {
        const aborted =
          (e instanceof DOMException && e.name === "AbortError") ||
          (e instanceof Error && e.name === "AbortError");
        if (aborted) {
          return;
        }
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant",
            content: "Something went wrong. Please try again.",
          },
        ]);
        return;
      } finally {
        setIsStreaming(false);
        abortRef.current = null;
      }

      const fieldsMatch = fullResponse.match(/<<<FIELDS>>>([\s\S]*?)<<<END>>>/);
      let displaySource = fullResponse;
      if (fieldsMatch && fieldsMatch[1] !== undefined) {
        try {
          const updates = JSON.parse(fieldsMatch[1].trim()) as Record<
            string,
            unknown
          >;
          displaySource = fullResponse
            .replace(/<<<FIELDS>>>[\s\S]*?<<<END>>>/, "")
            .trim();

          for (const key of WIZARD_KEYS) {
            if (!(key in updates)) continue;
            const v = updates[key];
            if (key === "starter_messages") {
              setFieldValue("starter_messages", normalizeStarters(v));
            } else {
              setFieldValue(key, v);
            }
            setFieldTouched(key, true, false);
            markFieldUpdated(key);
          }
        } catch (e) {
          console.error("Agent wizard field parse failed", e);
        }
      }

      setMessages((prev) => {
        const copy = [...prev];
        const last = copy[copy.length - 1];
        if (last && last.role === "assistant") {
          copy[copy.length - 1] = { ...last, content: displaySource };
        }
        return copy;
      });
    },
    [currentValues, markFieldUpdated, setFieldTouched, setFieldValue]
  );

  const handleInputBarSubmit = useCallback(
    (submittedMessage: string) => {
      const trimmed = submittedMessage.trim();
      if (!trimmed || isStreaming) return;
      chatInputBarRef.current?.reset();
      setMessages((prev) => {
        const userMessage: ChatMessage = { role: "user", content: trimmed };
        const nextMessages = [...prev, userMessage];
        void streamResponse(nextMessages);
        return nextMessages;
      });
    },
    [isStreaming, streamResponse]
  );

  const handleFileUploadNoop = useCallback((_files: File[]) => {
    /* attachments disabled in wizard */
  }, []);

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages, isStreaming]);

  return (
    <div className="flex h-full min-h-0 flex-col">
      {noLlmError && (
        <div className="border-b border-border-02 bg-background-tint-02 px-3 py-2">
          <Text font="secondary-body" color="text-03">
            Form-only mode: add an LLM under Settings to enable assistant chat.
          </Text>
        </div>
      )}
      <div
        ref={scrollRef}
        className="min-h-0 min-w-0 flex-1 overflow-y-auto overflow-x-hidden px-3 py-3"
      >
        <div className="mx-auto flex w-full min-w-0 max-w-full flex-col items-stretch gap-3">
          {messages.map((m, i) => (
            <div
              key={i}
              className={cn(
                THREAD_BUBBLE,
                "rounded-lg px-3 py-2",
                m.role === "user"
                  ? "self-end bg-background-tint-02"
                  : "self-start bg-background-neutral-02"
              )}
            >
              <Text font="main-ui-body" color="text-02">
                {m.content}
              </Text>
            </div>
          ))}
        </div>
      </div>
      <div className="min-w-0 border-t border-border-02 px-3 pb-3 pt-2">
        {!llmManager.isLoadingProviders && llmManager.hasAnyProvider && (
          <div className="pb-1">
            <ModelSelector
              llmManager={llmManager}
              selectedModels={multiModel.selectedModels}
              onAdd={multiModel.addModel}
              onRemove={multiModel.removeModel}
              onReplace={multiModel.replaceModel}
            />
          </div>
        )}
        {/* Breathing room so shadow-01 on AppInputBar is not clipped (see AppPage note on AppInputBar). */}
        <div className="h-[14px]" aria-hidden />
        <AppInputBar
          ref={chatInputBarRef}
          enableFileAttachments={false}
          enableVoiceInput={false}
          enablePromptShortcuts={false}
          chatPlaceholder="Describe the agent you want to build…"
          filterManager={filterManager}
          llmManager={llmManager}
          initialMessage=""
          stopGenerating={stopGenerating}
          onSubmit={handleInputBarSubmit}
          chatState={chatState}
          currentSessionFileTokenCount={0}
          availableContextTokens={AVAILABLE_CONTEXT_TOKENS}
          selectedAgent={WIZARD_INPUT_BAR_AGENT}
          handleFileUpload={handleFileUploadNoop}
          deepResearchEnabled={false}
          toggleDeepResearch={() => {}}
          disabled={
            !llmManager.isLoadingProviders && llmManager.hasAnyProvider === false
          }
        />
      </div>
    </div>
  );
}
