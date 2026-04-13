import React, { memo } from "react";
import { SvgArrowExchange } from "@opal/icons";

type ConnectionProviderIconProps = {
  icon: React.ReactNode;
};

const ConnectionProviderIcon = memo(({ icon }: ConnectionProviderIconProps) => {
  return (
    <div className="flex items-center gap-1">
      <div className="w-7 h-7 flex items-center justify-center">{icon}</div>
      <div className="w-4 h-4 flex items-center justify-center">
        <SvgArrowExchange className="w-3 h-3 stroke-text-04" />
      </div>
      <div className="w-7 h-7 flex items-center justify-center">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src="/meaningful-ai-icon.png"
          alt="Meaningful AI"
          width={24}
          height={24}
          className="opacity-50"
        />
      </div>
    </div>
  );
});

ConnectionProviderIcon.displayName = "ConnectionProviderIcon";

export default ConnectionProviderIcon;
