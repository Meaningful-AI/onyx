import type { ComponentProps } from "react";
import AgentPersonaForm from "@/refresh-pages/AgentPersonaForm";

export type { AgentEditorPageProps } from "@/refresh-pages/AgentPersonaForm";

export default function AgentEditorPage(
  props: Omit<ComponentProps<typeof AgentPersonaForm>, "variant">
) {
  return <AgentPersonaForm variant="editor" {...props} />;
}
