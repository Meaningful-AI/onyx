"use client";

/**
 * Settings Page Layout Components
 *
 * A namespaced collection of components for building consistent settings pages.
 * These components provide a standardized layout with scroll-aware headers,
 * centered content containers, and automatic responsive behavior.
 *
 * @example
 * ```tsx
 * import SettingsLayouts from "@/layouts/settings-layouts";
 * import { SvgSettings } from "@opal/icons";
 *
 * function MySettingsPage() {
 *   return (
 *     <SettingsLayouts.Root>
 *       <SettingsLayouts.Header
 *         icon={SvgSettings}
 *         title="Account Settings"
 *         description="Manage your account preferences and settings"
 *         rightChildren={<Button>Save</Button>}
 *       >
 *         <InputTypeIn placeholder="Search settings..." />
 *       </SettingsLayouts.Header>
 *
 *       <SettingsLayouts.Body>
 *         <Card>Settings content here</Card>
 *       </SettingsLayouts.Body>
 *     </SettingsLayouts.Root>
 *   );
 * }
 * ```
 */

import BackButton from "@/refresh-components/buttons/BackButton";
import { cn } from "@/lib/utils";
import Separator from "@/refresh-components/Separator";
import { WithoutStyles } from "@/types";
import { IconFunctionComponent } from "@opal/types";
import { HtmlHTMLAttributes, useEffect, useRef, useState } from "react";
import { Content } from "@opal/layouts";
import Spacer from "@/refresh-components/Spacer";

const widthClasses = {
  sm: "w-[min(var(--container-sm),100%)]",
  "sm-md": "w-[min(var(--container-sm-md),100%)]",
  md: "w-[min(var(--container-md),100%)]",
  lg: "w-[min(var(--container-lg),100%)]",
  full: "w-full min-w-0",
};

/**
 * Settings Root Component
 *
 * Wrapper component that provides the base structure for settings pages.
 * Creates a centered, scrollable container with configurable width.
 *
 * Features:
 * - Full height container with centered content
 * - Automatic overflow-y scrolling
 * - Contains the scroll container ID that Settings.Header uses for shadow detection
 * - Configurable width via CSS variables defined in sizes.css:
 *   "sm" (672px), "sm-md" (752px), "md" (872px, default), "lg" (992px), "full" (100%)
 *
 * @example
 * ```tsx
 * // Default medium width (872px max)
 * <SettingsLayouts.Root>
 *   <SettingsLayouts.Header {...} />
 *   <SettingsLayouts.Body>...</SettingsLayouts.Body>
 * </SettingsLayouts.Root>
 *
 * // Large width (992px max)
 * <SettingsLayouts.Root width="lg">
 *   <SettingsLayouts.Header {...} />
 *   <SettingsLayouts.Body>...</SettingsLayouts.Body>
 * </SettingsLayouts.Root>
 * ```
 */
interface SettingsRootProps
  extends WithoutStyles<React.HtmlHTMLAttributes<HTMLDivElement>> {
  width?: keyof typeof widthClasses;
  /** When false, the root does not scroll; children should own vertical overflow (e.g. split panes). */
  scrollable?: boolean;
}

interface SettingsBodyProps
  extends WithoutStyles<React.HtmlHTMLAttributes<HTMLDivElement>> {
  /** Tighter padding and gaps (e.g. agent wizard split-pane form column). */
  compact?: boolean;
}
function SettingsRoot({
  width = "md",
  scrollable = true,
  ...props
}: SettingsRootProps) {
  return (
    <div
      id="page-wrapper-scroll-container"
      className={cn(
        "flex min-h-0 w-full flex-col items-stretch",
        scrollable ? "h-full overflow-y-auto" : "h-full overflow-hidden"
      )}
    >
      {/* WARNING: The id="page-wrapper-scroll-container" above is used by SettingsHeader
          to detect scroll position and show/hide the scroll shadow.
          DO NOT REMOVE this ID without updating SettingsHeader accordingly. */}
      <div
        className={cn(
          "flex min-h-0 flex-col",
          scrollable ? "h-full" : "h-full min-h-0 flex-1",
          widthClasses[width]
        )}
      >
        <div
          className={cn(
            "flex min-h-0 flex-col",
            scrollable ? "h-full" : "h-full min-h-0 flex-1"
          )}
          {...props}
        />
      </div>
    </div>
  );
}

/**
 * Settings Header Component
 *
 * Sticky header component for settings pages with icon, title, description,
 * and optional actions. Automatically shows a scroll shadow when the page
 * has been scrolled down.
 *
 * Features:
 * - Sticky positioning at the top of the page
 * - Icon display (1.75rem size)
 * - Title (headingH2 style)
 * - Optional description (string)
 * - Optional right-aligned action buttons via rightChildren
 * - Optional children content below title/description
 * - Optional back button
 * - Optional bottom separator
 * - Automatic scroll shadow effect
 *
 * @example
 * ```tsx
 * // Basic header
 * <SettingsLayouts.Header
 *   icon={SvgUser}
 *   title="Profile Settings"
 *   description="Update your profile information"
 * />
 *
 * // Without description
 * <SettingsLayouts.Header
 *   icon={SvgUser}
 *   title="Profile Settings"
 * />
 *
 * // With action buttons
 * <SettingsLayouts.Header
 *   icon={SvgSettings}
 *   title="General Settings"
 *   description="Configure your preferences"
 *   rightChildren={
 *     <Button onClick={handleSave}>Save Changes</Button>
 *   }
 * />
 *
 * // With search/filter below and bottom separator
 * <SettingsLayouts.Header
 *   icon={SvgDatabase}
 *   title="Data Sources"
 *   description="Manage your connected data sources"
 *   separator
 * >
 *   <InputTypeIn placeholder="Search data sources..." />
 * </SettingsLayouts.Header>
 *
 * // With back button
 * <SettingsLayouts.Header
 *   icon={SvgArrow}
 *   title="Advanced Settings"
 *   description="Expert configuration options"
 *   backButton
 * />
 *
 * // With string description
 * <SettingsLayouts.Header
 *   icon={SvgDatabase}
 *   title="API Keys"
 *   description="Manage your API keys"
 * />
 * ```
 */
export interface SettingsHeaderProps {
  icon: IconFunctionComponent;
  title: string;
  description?: string;
  children?: React.ReactNode;
  rightChildren?: React.ReactNode;
  backButton?: boolean;
  onBack?: () => void;
  separator?: boolean;
  /** Tighter top spacing and title/back gap (e.g. split-pane wizard layouts). */
  dense?: boolean;
}
function SettingsHeader({
  icon: Icon,
  title,
  description,
  children,
  rightChildren,
  backButton,
  onBack,
  separator,
  dense,
}: SettingsHeaderProps) {
  const [showShadow, setShowShadow] = useState(false);
  const headerRef = useRef<HTMLDivElement>(null);

  // # NOTE (@Subash-Mohan)
  // Headers with actions are always sticky, others are not.
  const isSticky = !!rightChildren;

  useEffect(() => {
    if (!isSticky) return;

    // IMPORTANT: This component relies on SettingsRoot having the ID "page-wrapper-scroll-container"
    // on its scrollable container. If that ID is removed or changed, the scroll shadow will not work.
    const scrollContainer = document.getElementById(
      "page-wrapper-scroll-container"
    );
    if (!scrollContainer) return;

    const handleScroll = () => {
      // Show shadow if the scroll container has been scrolled down
      setShowShadow(scrollContainer.scrollTop > 0);
    };

    scrollContainer.addEventListener("scroll", handleScroll);
    handleScroll(); // Check initial state

    return () => scrollContainer.removeEventListener("scroll", handleScroll);
  }, [isSticky]);

  const compactToolbar =
    dense && backButton && rightChildren && children == null;

  const scrollShadow = isSticky ? (
    <div
      className={cn(
        "pointer-events-none absolute left-0 right-0 h-[0.5rem] rounded-b-08 opacity-0 transition-opacity duration-300",
        showShadow && "opacity-100"
      )}
      style={{
        background:
          "linear-gradient(to bottom, var(--mask-02), transparent)",
      }}
    />
  ) : null;

  if (compactToolbar) {
    return (
      <div
        ref={headerRef}
        className={cn(
          "w-full bg-background-tint-01",
          isSticky && "sticky top-0 z-settings-header"
        )}
      >
        <div className="flex min-h-0 w-full flex-wrap items-center gap-x-3 gap-y-2 p-4">
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="shrink-0">
              <BackButton behaviorOverride={onBack} />
            </div>
            <div
              aria-label="admin-page-title"
              className="min-w-0 flex-1 overflow-hidden"
            >
              <Content
                icon={Icon}
                title={title}
                description={description}
                sizePreset="headline"
                variant="section"
              />
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">{rightChildren}</div>
        </div>
        {separator ? <Separator noPadding /> : null}
        {scrollShadow}
      </div>
    );
  }

  return (
    <div
      ref={headerRef}
      className={cn(
        "w-full bg-background-tint-01",
        isSticky && "sticky top-0 z-settings-header",
        backButton && !dense && "md:pt-4"
      )}
    >
      {backButton && (
        <div className="px-2">
          <BackButton behaviorOverride={onBack} />
        </div>
      )}

      <Spacer vertical rem={dense ? 1 : 2.5} />

      <div className="flex flex-col gap-6 px-4">
        <div className="flex w-full justify-between">
          <div aria-label="admin-page-title">
            <Content
              icon={Icon}
              title={title}
              description={description}
              sizePreset="headline"
              variant="section"
            />
          </div>
          {rightChildren}
        </div>

        {children}
      </div>

      {separator ? (
        <>
          <Spacer vertical rem={dense ? 1 : 1.5} />
          <Separator noPadding />
        </>
      ) : (
        <Spacer vertical rem={dense ? 0.25 : 0.5} />
      )}

      {scrollShadow}
    </div>
  );
}

/**
 * Settings Body Component
 *
 * Content container for settings page body. Provides consistent padding
 * and vertical spacing for content sections.
 *
 * Features:
 * - Top padding: 1.5rem (pt-6)
 * - Bottom padding: 4.5rem (pb-[4.5rem])
 * - Horizontal padding: 1rem (px-4)
 * - Flex column layout with 2rem gap (gap-8)
 * - Full width container
 *
 * @example
 * ```tsx
 * <SettingsLayouts.Body>
 *   <Card>
 *     <h3>Section 1</h3>
 *     <p>Content here</p>
 *   </Card>
 *   <Card>
 *     <h3>Section 2</h3>
 *     <p>More content</p>
 *   </Card>
 * </SettingsLayouts.Body>
 * ```
 */
function SettingsBody({ compact, ...props }: SettingsBodyProps) {
  return (
    <div
      className={cn(
        "flex w-full flex-col",
        compact
          ? "min-h-0 gap-6 px-4 pb-8 pt-4"
          : "gap-8 px-4 pb-[4.5rem] pt-6"
      )}
      {...props}
    />
  );
}

export { SettingsRoot as Root, SettingsHeader as Header, SettingsBody as Body };
