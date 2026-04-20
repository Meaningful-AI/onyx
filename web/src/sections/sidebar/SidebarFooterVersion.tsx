"use client";

import type { Components } from "react-markdown";
import Text from "@/refresh-components/texts/Text";
import MinimalMarkdown from "@/components/chat/MinimalMarkdown";
import { useSettingsContext } from "@/providers/SettingsProvider";
import { APP_SLOGAN } from "@/lib/constants";
import { cn, ensureHrefProtocol } from "@/lib/utils";

const markdownComponents = {
  p: ({ children }) => (
    <Text as="p" text03 secondaryAction className="!my-0 text-left">
      {children}
    </Text>
  ),
  a: ({ href, className, children, ...rest }) => {
    const fullHref = ensureHrefProtocol(href);
    return (
      <a
        href={fullHref}
        target="_blank"
        rel="noopener noreferrer"
        {...rest}
        className={cn(className, "underline underline-offset-2")}
      >
        <Text text03 secondaryAction>
          {children}
        </Text>
      </a>
    );
  },
} satisfies Partial<Components>;

interface SidebarFooterVersionProps {
  /** When the sidebar strip is folded, hide the disclaimer to avoid clipping. */
  folded: boolean;
}

export default function SidebarFooterVersion({ folded }: SidebarFooterVersionProps) {
  const settings = useSettingsContext();

  const content =
    settings?.enterpriseSettings?.custom_lower_disclaimer_content ||
    `[Meaningful AI ${
      settings?.webVersion || "dev"
    }](https://www.meaningful-ai.com/) - ${APP_SLOGAN}`;

  if (folded) {
    return null;
  }

  return (
    <div className="border-t border-border-02 pt-2 pb-1">
      <MinimalMarkdown
        content={content}
        className="max-w-full [&_*]:text-left"
        components={markdownComponents}
      />
    </div>
  );
}
