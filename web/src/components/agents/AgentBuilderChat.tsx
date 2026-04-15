"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import { useFormikContext } from "formik";
import { SvgArrowUp } from "@opal/icons";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ChatMessage {
  role: "assistant" | "user";
  content: string;
}

interface AgentBuilderChatProps {
  onFieldsUpdated?: (fieldNames: string[]) => void;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const WELCOME_MESSAGE: ChatMessage = {
  role: "assistant",
  content:
    "Tell me about the agent you want to create. What should it do? Who is it for?",
};

const FIELDS_START = "<<<FIELDS>>>";
const FIELDS_END = "<<<END>>>";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Strip any <<<FIELDS>>>...<<<END>>> block from visible text. */
function stripFieldsBlock(text: string): string {
  const startIdx = text.indexOf(FIELDS_START);
  if (startIdx === -1) return text;
  const endIdx = text.indexOf(FIELDS_END, startIdx);
  if (endIdx === -1) {
    // Block started but not finished — hide from the start marker onward
    return text.slice(0, startIdx).trimEnd();
  }
  return (
    text.slice(0, startIdx) + text.slice(endIdx + FIELDS_END.length)
  ).trimEnd();
}

/** Extract the JSON between <<<FIELDS>>> and <<<END>>>. */
function extractFieldsJson(text: string): Record<string, unknown> | null {
  const startIdx = text.indexOf(FIELDS_START);
  if (startIdx === -1) return null;
  const endIdx = text.indexOf(FIELDS_END, startIdx);
  if (endIdx === -1) return null;
  const jsonStr = text.slice(startIdx + FIELDS_START.length, endIdx).trim();
  try {
    return JSON.parse(jsonStr) as Record<string, unknown>;
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function AgentBuilderChat({
  onFieldsUpdated,
}: AgentBuilderChatProps) {
  const { values, setFieldValue } = useFormikContext<Record<string, unknown>>();

  const [messages, setMessages] = useState<ChatMessage[]>([WELCOME_MESSAGE]);
  const [input, setInput] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Auto-resize textarea
  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = `${Math.min(ta.scrollHeight, 120)}px`;
  }, [input]);

  // ------------------------------------------------------------------
  // Send message
  // ------------------------------------------------------------------

  const handleSend = useCallback(async () => {
    const trimmed = input.trim();
    if (!trimmed || isStreaming) return;

    const userMessage: ChatMessage = { role: "user", content: trimmed };
    const updatedMessages = [...messages, userMessage];
    setMessages(updatedMessages);
    setInput("");
    setIsStreaming(true);

    // Gather current form values to send along
    const currentValues: Record<string, unknown> = {};
    const formFields = [
      "name",
      "description",
      "instructions",
      "starter_messages",
      "web_search",
      "image_generation",
      "code_interpreter",
    ];
    for (const field of formFields) {
      currentValues[field] = values[field];
    }

    // Placeholder for assistant response
    const assistantIdx = updatedMessages.length;
    setMessages((prev) => [...prev, { role: "assistant", content: "" }]);

    let fullText = "";

    try {
      const response = await fetch("/api/agent-wizard-chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          messages: updatedMessages.map((m) => ({
            role: m.role,
            content: m.content,
          })),
          currentValues,
        }),
      });

      if (!response.ok) {
        const errorBody = await response.text().catch(() => "Unknown error");
        setMessages((prev) => {
          const copy = [...prev];
          copy[assistantIdx] = {
            role: "assistant",
            content: `Sorry, something went wrong: ${errorBody}. You can still fill in the form directly.`,
          };
          return copy;
        });
        setIsStreaming(false);
        return;
      }

      const provider = response.headers.get("X-LLM-Provider") ?? "";
      const reader = response.body?.getReader();
      if (!reader) throw new Error("No response body");

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // Process complete SSE lines
        const lines = buffer.split("\n");
        // Keep the last potentially incomplete line in the buffer
        buffer = lines.pop() ?? "";

        for (const line of lines) {
          const trimmedLine = line.trim();
          if (!trimmedLine || trimmedLine === "data: [DONE]") continue;
          if (!trimmedLine.startsWith("data: ")) continue;

          const jsonStr = trimmedLine.slice(6);
          let delta = "";

          try {
            const parsed = JSON.parse(jsonStr);

            if (provider === "anthropic") {
              // Anthropic format: content_block_delta
              if (
                parsed.type === "content_block_delta" &&
                parsed.delta?.text
              ) {
                delta = parsed.delta.text;
              }
            } else if (provider === "openai") {
              // OpenAI format: choices[0].delta.content
              if (parsed.choices?.[0]?.delta?.content) {
                delta = parsed.choices[0].delta.content;
              }
            } else {
              // Flexible: try both formats
              if (
                parsed.type === "content_block_delta" &&
                parsed.delta?.text
              ) {
                delta = parsed.delta.text;
              } else if (parsed.choices?.[0]?.delta?.content) {
                delta = parsed.choices[0].delta.content;
              }
            }
          } catch {
            // Skip unparseable lines
          }

          if (delta) {
            fullText += delta;
            const displayText = stripFieldsBlock(fullText);
            setMessages((prev) => {
              const copy = [...prev];
              copy[assistantIdx] = {
                role: "assistant",
                content: displayText,
              };
              return copy;
            });
          }
        }
      }

      // Stream complete — final parse for fields
      const fieldsData = extractFieldsJson(fullText);
      const displayText = stripFieldsBlock(fullText);

      setMessages((prev) => {
        const copy = [...prev];
        copy[assistantIdx] = { role: "assistant", content: displayText };
        return copy;
      });

      if (fieldsData) {
        const updatedFieldNames: string[] = [];
        for (const [key, val] of Object.entries(fieldsData)) {
          if (formFields.includes(key)) {
            setFieldValue(key, val);
            updatedFieldNames.push(key);
          }
        }
        if (updatedFieldNames.length > 0) {
          onFieldsUpdated?.(updatedFieldNames);
        }
      }
    } catch {
      setMessages((prev) => {
        const copy = [...prev];
        copy[assistantIdx] = {
          role: "assistant",
          content:
            "Sorry, I couldn't connect. You can still fill in the form directly.",
        };
        return copy;
      });
    } finally {
      setIsStreaming(false);
    }
  }, [input, isStreaming, messages, values, setFieldValue, onFieldsUpdated]);

  // ------------------------------------------------------------------
  // Key handler
  // ------------------------------------------------------------------

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  // ------------------------------------------------------------------
  // Render
  // ------------------------------------------------------------------

  return (
    <div className="flex flex-col h-full bg-background border-l border-border">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-base font-semibold text-text">Agent Builder</h2>
        <p className="text-xs text-text-muted">
          Describe your agent and I&apos;ll help configure it
        </p>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
        {messages.map((msg, idx) => (
          <div
            key={idx}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            {msg.role === "assistant" && (
              <div className="flex-shrink-0 w-6 h-6 rounded-full bg-accent flex items-center justify-center mr-2 mt-0.5">
                <span className="text-[10px] font-bold text-white">AI</span>
              </div>
            )}
            <div
              className={`max-w-[80%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap ${
                msg.role === "user"
                  ? "bg-background-emphasis text-text"
                  : "bg-background-subtle text-text"
              }`}
            >
              {msg.content ||
                (isStreaming && idx === messages.length - 1 ? (
                  <TypingIndicator />
                ) : null)}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="px-4 py-3 border-t border-border">
        <div className="flex items-end gap-2">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Describe your agent..."
            disabled={isStreaming}
            rows={1}
            className="flex-1 resize-none rounded-lg border border-border bg-background px-3 py-2 text-sm text-text placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
            style={{ maxHeight: 120 }}
          />
          <button
            onClick={handleSend}
            disabled={isStreaming || !input.trim()}
            className="flex-shrink-0 w-8 h-8 rounded-lg bg-accent flex items-center justify-center text-white disabled:opacity-40 hover:opacity-90 transition-opacity"
            aria-label="Send message"
          >
            <SvgArrowUp className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// TypingIndicator
// ---------------------------------------------------------------------------

function TypingIndicator() {
  return (
    <div className="flex items-center gap-1 py-1">
      <span className="w-1.5 h-1.5 rounded-full bg-text-muted animate-bounce [animation-delay:0ms]" />
      <span className="w-1.5 h-1.5 rounded-full bg-text-muted animate-bounce [animation-delay:150ms]" />
      <span className="w-1.5 h-1.5 rounded-full bg-text-muted animate-bounce [animation-delay:300ms]" />
    </div>
  );
}
