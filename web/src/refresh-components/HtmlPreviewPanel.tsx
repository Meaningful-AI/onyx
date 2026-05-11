"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import Text from "@/refresh-components/texts/Text";
import { Button } from "@opal/components/buttons/button/components";
import { SvgExternalLink, SvgX } from "@opal/icons";
import {
  useCurrentHtmlPreview,
  useChatSessionStore,
} from "@/app/app/stores/useChatSessionStore";
import { fetchChatFile } from "@/lib/chat/svc";

function useHtmlContent(): { content: string | null; isLoading: boolean } {
  const htmlPreview = useCurrentHtmlPreview();
  const [fetchedContent, setFetchedContent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Derive a stable string key so the effect only fires when the source
  // actually changes identity, not on every render due to object reference churn.
  const sourceKey =
    htmlPreview.source === null
      ? null
      : htmlPreview.source.kind === "content"
        ? `content:${htmlPreview.source.content}`
        : `file:${htmlPreview.source.fileId}`;

  const prevKeyRef = useRef<string | null | undefined>(undefined);

  useEffect(() => {
    if (prevKeyRef.current === sourceKey) return;
    prevKeyRef.current = sourceKey;

    if (!htmlPreview.source) {
      setFetchedContent(null);
      return;
    }
    if (htmlPreview.source.kind === "content") {
      setFetchedContent(htmlPreview.source.content);
      return;
    }
    const { fileId } = htmlPreview.source;
    setIsLoading(true);
    setFetchedContent(null);
    fetchChatFile(fileId)
      .then((res) => res.text())
      .then((text) => setFetchedContent(text))
      .catch(() => setFetchedContent(null))
      .finally(() => setIsLoading(false));
    // sourceKey is the stable primitive — htmlPreview.source is intentionally
    // NOT listed so we avoid object-reference churn triggering the effect.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sourceKey]);

  return { content: fetchedContent, isLoading };
}

export default function HtmlPreviewPanel() {
  const htmlPreview = useCurrentHtmlPreview();
  const updateCurrentHtmlPreview = useChatSessionStore(
    (s) => s.updateCurrentHtmlPreview
  );
  const { content, isLoading } = useHtmlContent();

  function handleClose() {
    updateCurrentHtmlPreview({ visible: false });
  }

  function handleOpenInNewTab() {
    if (!content) return;
    const blob = new Blob([content], { type: "text/html" });
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank", "noopener,noreferrer");
  }

  return (
    <div
      className={cn(
        "flex flex-col h-full border-l border-border-02 bg-background-neutral-01"
      )}
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-border-02 flex-shrink-0">
        <Text as="p" secondaryAction>
          {htmlPreview.title || "HTML Preview"}
        </Text>
        <div className="flex items-center gap-1">
          <Button
            prominence="tertiary"
            size="xs"
            icon={SvgExternalLink}
            onClick={handleOpenInNewTab}
            disabled={!content}
            tooltip="Open in new tab"
          />
          <Button
            prominence="tertiary"
            size="xs"
            icon={SvgX}
            onClick={handleClose}
            tooltip="Close preview"
          />
        </div>
      </div>

      <div className="flex-1 overflow-hidden">
        {isLoading && (
          <div className="flex items-center justify-center h-full">
            <Text as="p" secondaryBody text03>
              Loading…
            </Text>
          </div>
        )}
        {!isLoading && content && (
          <iframe
            srcDoc={content}
            sandbox="allow-scripts allow-modals allow-popups"
            className="w-full h-full border-0"
            title={htmlPreview.title || "HTML Preview"}
          />
        )}
        {!isLoading && !content && htmlPreview.source && (
          <div className="flex items-center justify-center h-full">
            <Text as="p" secondaryBody text03>
              Failed to load preview.
            </Text>
          </div>
        )}
      </div>
    </div>
  );
}
