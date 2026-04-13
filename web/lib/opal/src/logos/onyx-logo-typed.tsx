import { cn } from "@opal/utils";

interface OnyxLogoTypedProps {
  size?: number;
  className?: string;
}

const SvgOnyxLogoTyped = ({ size: height, className }: OnyxLogoTypedProps) => {
  // The SVG logo is white, suitable for dark sidebar backgrounds.
  // Scale width proportionally (logo aspect ratio is roughly 4:1).
  const width = height != null ? height * 4 : undefined;

  return (
    <div
      className={cn("flex flex-row items-center gap-2", className)}
    >
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src="/meaningful-ai-icon.png"
        alt="Meaningful AI"
        width={height}
        height={height}
        style={{ width: height, height: height }}
      />
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src="/meaningful-ai-logo.svg"
        alt="Meaningful AI"
        height={height}
        style={{ height: height, width: "auto" }}
      />
    </div>
  );
};
export default SvgOnyxLogoTyped;
