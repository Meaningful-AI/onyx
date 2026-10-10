"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "@/hooks/useToast";

export default function EEFeatureRedirect() {
  const router = useRouter();

  useEffect(() => {
    toast.error(
      "This feature requires an active license. Contact your administrator to enable access."
    );
    router.replace("/app");
  }, [router]);

  return null;
}
