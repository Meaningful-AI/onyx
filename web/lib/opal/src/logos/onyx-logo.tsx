import type { IconProps } from "@opal/types";
const SvgOnyxLogo = ({ size, className, style, ...rest }: IconProps) => {
  const resolvedStyle = {
    ...(style ?? {}),
    width: size ?? style?.width,
    height: size ?? style?.height,
    flexShrink: 0,
  };
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src="/meaningful-ai-icon.png"
      alt="Meaningful AI"
      width={typeof resolvedStyle.width === "number" ? resolvedStyle.width : undefined}
      height={typeof resolvedStyle.height === "number" ? resolvedStyle.height : undefined}
      style={resolvedStyle as React.CSSProperties}
      className={className}
    />
  );
};
export default SvgOnyxLogo;
