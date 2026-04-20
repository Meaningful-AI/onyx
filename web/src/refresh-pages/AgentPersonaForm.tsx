"use client";

import { useState, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import * as SettingsLayouts from "@/layouts/settings-layouts";
import * as GeneralLayouts from "@/layouts/general-layouts";
import { Button as OpalButton } from "@opal/components";
import { FullPersona } from "@/app/admin/agents/interfaces";
import { Formik, Form } from "formik";
import * as Yup from "yup";
import { parseLlmDescriptor, structureValue } from "@/lib/llmConfig/utils";
import { useLLMProviders } from "@/hooks/useLLMProviders";
import {
  STARTER_MESSAGES_EXAMPLES,
  MAX_CHARACTERS_STARTER_MESSAGE,
  MAX_CHARACTERS_AGENT_DESCRIPTION,
} from "@/lib/constants";
import {
  IMAGE_GENERATION_TOOL_ID,
  WEB_SEARCH_TOOL_ID,
  PYTHON_TOOL_ID,
  SEARCH_TOOL_ID,
  OPEN_URL_TOOL_ID,
} from "@/app/app/components/tools/constants";
import Text from "@/refresh-components/texts/Text";
import SimpleTooltip from "@/refresh-components/SimpleTooltip";
import { useDocumentSets } from "@/app/admin/documents/sets/hooks";
import { useProjectsContext } from "@/providers/ProjectsContext";
import { useCreateModal } from "@/refresh-components/contexts/ModalContext";
import { toast } from "@/hooks/useToast";
import UserFilesModal from "@/components/modals/UserFilesModal";
import {
  ProjectFile,
  UserFileStatus,
} from "@/app/app/projects/projectsService";
import { SvgOnyxOctagon, SvgTrash } from "@opal/icons";
import { useAgents } from "@/hooks/useAgents";
import {
  createPersona,
  updatePersona,
  PersonaUpsertParameters,
} from "@/app/admin/agents/lib";
import useMcpServersForAgentEditor from "@/hooks/useMcpServersForAgentEditor";
import useOpenApiTools from "@/hooks/useOpenApiTools";
import { useAvailableTools } from "@/hooks/useAvailableTools";
import { getActionIcon } from "@/lib/tools/mcpUtils";
import { MCPServer, MCPTool, ToolSnapshot } from "@/lib/tools/interfaces";
import { useAppRouter } from "@/hooks/appNavigation";
import { isDateInFuture } from "@/lib/dateUtils";
import {
  deleteAgent,
  updateAgentFeaturedStatus,
  updateAgentSharedStatus,
} from "@/lib/agents";
import ConfirmationModalLayout from "@/refresh-components/layouts/ConfirmationModalLayout";
import ShareAgentModal from "@/sections/modals/ShareAgentModal";
import { ValidSources } from "@/lib/types";
import { useVectorDbEnabled } from "@/providers/SettingsProvider";
import { useUser } from "@/providers/UserProvider";
import { usePaidEnterpriseFeaturesEnabled } from "@/components/settings/usePaidEnterpriseFeaturesEnabled";
import { cn } from "@/lib/utils";
import AgentFormBody from "@/components/agents/AgentFormBody";
import AgentBuilderChat from "@/sections/agents/AgentBuilderChat";

export interface AgentPersonaFormProps {
  agent?: FullPersona;
  refreshAgent?: () => void;
  variant?: "editor" | "wizard";
}

/** @deprecated Use AgentPersonaFormProps */
export type AgentEditorPageProps = AgentPersonaFormProps;

export default function AgentPersonaForm({
  agent: existingAgent,
  refreshAgent,
  variant = "editor",
}: AgentPersonaFormProps) {
  const router = useRouter();
  const [highlightedFields, setHighlightedFields] = useState<Set<string>>(
    () => new Set()
  );
  const highlightTimeoutsRef = useRef<
    Map<string, ReturnType<typeof setTimeout>>
  >(new Map());

  function markFieldUpdated(key: string) {
    setHighlightedFields((prev) => new Set(prev).add(key));
    const existing = highlightTimeoutsRef.current.get(key);
    if (existing) {
      clearTimeout(existing);
    }
    const t = setTimeout(() => {
      setHighlightedFields((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
      highlightTimeoutsRef.current.delete(key);
    }, 2000);
    highlightTimeoutsRef.current.set(key, t);
  }
  const appRouter = useAppRouter();
  const { refresh: refreshAgents } = useAgents();
  const shareAgentModal = useCreateModal();
  const deleteAgentModal = useCreateModal();
  const { isAdmin, isCurator } = useUser();
  const canUpdateFeaturedStatus = isAdmin || isCurator;
  const vectorDbEnabled = useVectorDbEnabled();
  const isPaidEnterpriseFeaturesEnabled = usePaidEnterpriseFeaturesEnabled();

  // LLM Model Selection
  const getCurrentLlm = useCallback(
    (values: any, llmProviders: any) =>
      values.llm_model_version_override && values.llm_model_provider_override
        ? (() => {
            const provider = llmProviders?.find(
              (p: any) => p.name === values.llm_model_provider_override
            );
            return structureValue(
              values.llm_model_provider_override,
              provider?.provider || "",
              values.llm_model_version_override
            );
          })()
        : null,
    []
  );

  const onLlmSelect = useCallback(
    (selected: string | null, setFieldValue: any) => {
      if (selected === null) {
        setFieldValue("llm_model_version_override", null);
        setFieldValue("llm_model_provider_override", null);
      } else {
        const { modelName, name } = parseLlmDescriptor(selected);
        if (modelName && name) {
          setFieldValue("llm_model_version_override", modelName);
          setFieldValue("llm_model_provider_override", name);
        }
      }
    },
    []
  );

  // Hooks for Knowledge section
  const { allRecentFiles, beginUpload } = useProjectsContext();
  const { data: documentSets } = useDocumentSets();
  const userFilesModal = useCreateModal();
  const [presentingDocument, setPresentingDocument] = useState<{
    document_id: string;
    semantic_identifier: string;
  } | null>(null);

  const { mcpData, isLoading: isMcpLoading } = useMcpServersForAgentEditor();
  const { openApiTools: openApiToolsRaw, isLoading: isOpenApiLoading } =
    useOpenApiTools();
  const { llmProviders } = useLLMProviders(existingAgent?.id);
  const mcpServers = mcpData?.mcp_servers ?? [];
  const openApiTools = openApiToolsRaw ?? [];

  // Check if the *BUILT-IN* tools are available.
  // The built-in tools are:
  // - image-gen
  // - web-search
  // - code-interpreter
  const { tools: availableTools, isLoading: isToolsLoading } =
    useAvailableTools();
  const searchTool = availableTools?.find(
    (t) => t.in_code_tool_id === SEARCH_TOOL_ID
  );
  const imageGenTool = availableTools?.find(
    (t) => t.in_code_tool_id === IMAGE_GENERATION_TOOL_ID
  );
  const webSearchTool = availableTools?.find(
    (t) => t.in_code_tool_id === WEB_SEARCH_TOOL_ID
  );
  const openURLTool = availableTools?.find(
    (t) => t.in_code_tool_id === OPEN_URL_TOOL_ID
  );
  const codeInterpreterTool = availableTools?.find(
    (t) => t.in_code_tool_id === PYTHON_TOOL_ID
  );
  const isImageGenerationAvailable = !!imageGenTool;
  const imageGenerationDisabledTooltip = isImageGenerationAvailable
    ? undefined
    : "Image generation requires a configured model. If you have access, set one up under Settings > Image Generation, or ask an admin.";

  // Group MCP server tools from availableTools by server ID
  const mcpServersWithTools = mcpServers.map((server) => {
    const serverTools: MCPTool[] = (availableTools || [])
      .filter((tool) => tool.mcp_server_id === server.id)
      .map((tool) => ({
        id: tool.id.toString(),
        icon: getActionIcon(server.server_url, server.name),
        name: tool.display_name || tool.name,
        description: tool.description,
        isAvailable: true,
        isEnabled: tool.enabled,
      }));

    return { server, tools: serverTools, isLoading: false };
  });

  const initialValues = {
    // General
    icon_name: existingAgent?.icon_name ?? null,
    uploaded_image_id: existingAgent?.uploaded_image_id ?? null,
    remove_image: false,
    name: existingAgent?.name ?? "",
    description: existingAgent?.description ?? "",

    // Prompts
    instructions: existingAgent?.system_prompt ?? "",
    starter_messages: Array.from(
      { length: STARTER_MESSAGES_EXAMPLES.length },
      (_, i) => existingAgent?.starter_messages?.[i]?.message ?? ""
    ),

    // Knowledge - enabled if agent has any knowledge sources attached
    enable_knowledge:
      (existingAgent?.document_sets?.length ?? 0) > 0 ||
      (existingAgent?.hierarchy_nodes?.length ?? 0) > 0 ||
      (existingAgent?.attached_documents?.length ?? 0) > 0 ||
      (existingAgent?.user_file_ids?.length ?? 0) > 0,
    document_set_ids: existingAgent?.document_sets?.map((ds) => ds.id) ?? [],
    // Individual document IDs from hierarchy browsing
    document_ids: existingAgent?.attached_documents?.map((doc) => doc.id) ?? [],
    // Hierarchy node IDs (folders/spaces/channels) for scoped search
    hierarchy_node_ids:
      existingAgent?.hierarchy_nodes?.map((node) => node.id) ?? [],
    user_file_ids: existingAgent?.user_file_ids ?? [],
    // Selected sources for the new knowledge UI - derived from document sets
    selected_sources: [] as ValidSources[],

    // Advanced
    llm_model_provider_override:
      existingAgent?.llm_model_provider_override ?? null,
    llm_model_version_override:
      existingAgent?.llm_model_version_override ?? null,
    knowledge_cutoff_date: existingAgent?.search_start_date
      ? new Date(existingAgent.search_start_date)
      : null,
    replace_base_system_prompt:
      existingAgent?.replace_base_system_prompt ?? false,
    reminders: existingAgent?.task_prompt ?? "",
    // For new agents, default to false for optional tools to avoid
    // "Tool not available" errors when the tool isn't configured.
    // For existing agents, preserve the current tool configuration.
    image_generation:
      !!imageGenTool &&
      (existingAgent?.tools?.some(
        (tool) => tool.in_code_tool_id === IMAGE_GENERATION_TOOL_ID
      ) ??
        false),
    web_search:
      !!webSearchTool &&
      (existingAgent?.tools?.some(
        (tool) => tool.in_code_tool_id === WEB_SEARCH_TOOL_ID
      ) ??
        false),
    open_url:
      !!openURLTool &&
      (existingAgent?.tools?.some(
        (tool) => tool.in_code_tool_id === OPEN_URL_TOOL_ID
      ) ??
        false),
    code_interpreter:
      !!codeInterpreterTool &&
      (existingAgent?.tools?.some(
        (tool) => tool.in_code_tool_id === PYTHON_TOOL_ID
      ) ??
        false),
    // MCP servers - dynamically add fields for each server with nested tool fields
    ...Object.fromEntries(
      mcpServersWithTools.map(({ server, tools }) => {
        // Find all tools from existingAgent that belong to this MCP server
        const serverToolsFromAgent =
          existingAgent?.tools?.filter(
            (tool) => tool.mcp_server_id === server.id
          ) ?? [];

        // Build the tool field object with tool_{id} for ALL available tools
        const toolFields: Record<string, boolean> = {};
        tools.forEach((tool) => {
          // Set to true if this tool was enabled in existingAgent, false otherwise
          toolFields[`tool_${tool.id}`] = serverToolsFromAgent.some(
            (t) => t.id === Number(tool.id)
          );
        });

        return [
          `mcp_server_${server.id}`,
          {
            enabled: serverToolsFromAgent.length > 0, // Server is enabled if it has any tools
            ...toolFields, // Add individual tool states for ALL tools
          },
        ];
      })
    ),

    // OpenAPI tools - add a boolean field for each tool
    ...Object.fromEntries(
      openApiTools.map((openApiTool) => [
        `openapi_tool_${openApiTool.id}`,
        existingAgent?.tools?.some((t) => t.id === openApiTool.id) ?? false,
      ])
    ),

    // Sharing
    shared_user_ids: existingAgent?.users?.map((user) => user.id) ?? [],
    shared_group_ids: existingAgent?.groups ?? [],
    is_public: existingAgent?.is_public ?? false,
    label_ids: existingAgent?.labels?.map((l) => l.id) ?? [],
    is_featured: existingAgent?.is_featured ?? false,
  };

  const validationSchema = Yup.object().shape({
    // General
    icon_name: Yup.string().nullable(),
    remove_image: Yup.boolean().optional(),
    uploaded_image_id: Yup.string().nullable(),
    name: Yup.string().required("Agent name is required."),
    description: Yup.string()
      .max(
        MAX_CHARACTERS_AGENT_DESCRIPTION,
        `Description must be ${MAX_CHARACTERS_AGENT_DESCRIPTION} characters or less`
      )
      .optional(),

    // Prompts
    instructions: Yup.string().optional(),
    starter_messages: Yup.array().of(
      Yup.string().max(
        MAX_CHARACTERS_STARTER_MESSAGE,
        `Conversation starter must be ${MAX_CHARACTERS_STARTER_MESSAGE} characters or less`
      )
    ),

    // Knowledge
    enable_knowledge: Yup.boolean(),
    document_set_ids: Yup.array().of(Yup.number()),
    document_ids: Yup.array().of(Yup.string()),
    hierarchy_node_ids: Yup.array().of(Yup.number()),
    user_file_ids: Yup.array().of(Yup.string()),
    selected_sources: Yup.array().of(Yup.string()),

    // Advanced
    llm_model_provider_override: Yup.string().nullable().optional(),
    llm_model_version_override: Yup.string().nullable().optional(),
    knowledge_cutoff_date: Yup.date()
      .nullable()
      .optional()
      .test(
        "knowledge-cutoff-date-not-in-future",
        "Knowledge cutoff date must be today or earlier.",
        (value) => !value || !isDateInFuture(value)
      ),
    replace_base_system_prompt: Yup.boolean(),
    reminders: Yup.string().optional(),

    // MCP servers - dynamically add validation for each server with nested tool validation
    ...Object.fromEntries(
      mcpServers.map((server) => [
        `mcp_server_${server.id}`,
        Yup.object(), // Allow any nested tool fields as booleans
      ])
    ),

    // OpenAPI tools - add boolean validation for each tool
    ...Object.fromEntries(
      openApiTools.map((openApiTool) => [
        `openapi_tool_${openApiTool.id}`,
        Yup.boolean(),
      ])
    ),
  });

  async function handleSubmit(values: typeof initialValues) {
    try {
      // Map conversation starters
      const starterMessages = values.starter_messages
        .filter((message: string) => message.trim() !== "")
        .map((message: string) => ({
          message: message,
          name: message,
        }));

      // Send null instead of empty array if no starter messages
      const finalStarterMessages =
        starterMessages.length > 0 ? starterMessages : null;

      // Always look up tools in availableTools to ensure we can find all tools

      const toolIds = [];
      if (values.enable_knowledge) {
        if (vectorDbEnabled && searchTool) {
          toolIds.push(searchTool.id);
        }
      }
      if (values.image_generation && imageGenTool) {
        toolIds.push(imageGenTool.id);
      }
      if (values.web_search && webSearchTool) {
        toolIds.push(webSearchTool.id);
      }
      if (values.open_url && openURLTool) {
        toolIds.push(openURLTool.id);
      }
      if (values.code_interpreter && codeInterpreterTool) {
        toolIds.push(codeInterpreterTool.id);
      }

      // Collect enabled MCP tool IDs
      mcpServers.forEach((server) => {
        const serverFieldName = `mcp_server_${server.id}`;
        const serverData = (values as any)[serverFieldName];

        if (
          serverData &&
          typeof serverData === "object" &&
          serverData.enabled
        ) {
          // Server is enabled, collect all enabled tools
          Object.keys(serverData).forEach((key) => {
            if (key.startsWith("tool_") && serverData[key] === true) {
              // Extract tool ID from key (e.g., "tool_123" -> 123)
              const toolId = parseInt(key.replace("tool_", ""), 10);
              if (!isNaN(toolId)) {
                toolIds.push(toolId);
              }
            }
          });
        }
      });

      // Collect enabled OpenAPI tool IDs
      openApiTools.forEach((openApiTool) => {
        const toolFieldName = `openapi_tool_${openApiTool.id}`;
        if ((values as any)[toolFieldName] === true) {
          toolIds.push(openApiTool.id);
        }
      });

      // Build submission data
      const submissionData: PersonaUpsertParameters = {
        name: values.name,
        description: values.description,
        document_set_ids: values.enable_knowledge
          ? values.document_set_ids
          : [],
        is_public: values.is_public,
        llm_model_provider_override: values.llm_model_provider_override || null,
        llm_model_version_override: values.llm_model_version_override || null,
        starter_messages: finalStarterMessages,
        users: values.shared_user_ids,
        groups: values.shared_group_ids,
        tool_ids: toolIds,
        // uploaded_image: null, // Already uploaded separately
        remove_image: values.remove_image ?? false,
        uploaded_image_id: values.uploaded_image_id,
        icon_name: values.icon_name,
        search_start_date: values.knowledge_cutoff_date || null,
        label_ids: values.label_ids,
        is_featured: values.is_featured,
        // display_priority: ...,

        user_file_ids: values.enable_knowledge ? values.user_file_ids : [],
        hierarchy_node_ids: values.enable_knowledge
          ? values.hierarchy_node_ids
          : [],
        document_ids: values.enable_knowledge ? values.document_ids : [],

        system_prompt: values.instructions,
        replace_base_system_prompt: values.replace_base_system_prompt,
        task_prompt: values.reminders || "",
        datetime_aware: false,
      };

      // Call API
      let personaResponse;
      if (!!existingAgent) {
        personaResponse = await updatePersona(existingAgent.id, submissionData);
      } else {
        personaResponse = await createPersona(submissionData);
      }

      // Handle response
      if (!personaResponse || !personaResponse.ok) {
        const error = personaResponse
          ? await personaResponse.text()
          : "No response received";
        toast.error(
          `Failed to ${existingAgent ? "update" : "create"} agent - ${error}`
        );
        return;
      }

      // Success
      const agent = await personaResponse.json();
      toast.success(
        `Agent "${agent.name}" ${
          existingAgent ? "updated" : "created"
        } successfully`
      );

      // Refresh agents list and the specific agent
      await refreshAgents();
      if (refreshAgent) {
        refreshAgent();
      }

      // Immediately start a chat with this agent.
      appRouter({ agentId: agent.id });
    } catch (error) {
      console.error("Submit error:", error);
      toast.error(`An error occurred: ${error}`);
    }
  }

  // Delete agent handler
  async function handleDeleteAgent() {
    if (!existingAgent) return;

    const error = await deleteAgent(existingAgent.id);

    if (error) {
      toast.error(`Failed to delete agent: ${error}`);
    } else {
      toast.success("Agent deleted successfully");

      deleteAgentModal.toggle(false);
      await refreshAgents();
      router.push("/app/agents");
    }
  }

  // FilePickerPopover callbacks for Knowledge section
  function handlePickRecentFile(
    file: ProjectFile,
    currentFileIds: string[],
    setFieldValue: (field: string, value: unknown) => void
  ) {
    if (!currentFileIds.includes(file.id)) {
      setFieldValue("user_file_ids", [...currentFileIds, file.id]);
    }
  }

  function handleUnpickRecentFile(
    file: ProjectFile,
    currentFileIds: string[],
    setFieldValue: (field: string, value: unknown) => void
  ) {
    setFieldValue(
      "user_file_ids",
      currentFileIds.filter((id) => id !== file.id)
    );
  }

  function handleFileClick(file: ProjectFile) {
    setPresentingDocument({
      document_id: `project_file__${file.file_id}`,
      semantic_identifier: file.name,
    });
  }

  async function handleUploadChange(
    e: React.ChangeEvent<HTMLInputElement>,
    currentFileIds: string[],
    setFieldValue: (field: string, value: unknown) => void
  ) {
    const files = e.target.files;
    if (!files || files.length === 0) return;
    try {
      let selectedIds = [...(currentFileIds || [])];
      const optimistic = await beginUpload(
        Array.from(files),
        null,
        (result) => {
          const uploadedFiles = result.user_files || [];
          if (uploadedFiles.length === 0) return;
          const tempToFinal = new Map(
            uploadedFiles
              .filter((f) => f.temp_id)
              .map((f) => [f.temp_id as string, f.id])
          );
          const replaced = (selectedIds || []).map(
            (id: string) => tempToFinal.get(id) ?? id
          );
          selectedIds = replaced;
          setFieldValue("user_file_ids", replaced);
        }
      );
      if (optimistic) {
        const optimisticIds = optimistic.map((f) => f.id);
        selectedIds = [...selectedIds, ...optimisticIds];
        setFieldValue("user_file_ids", selectedIds);
      }
    } catch (error) {
      console.error("Upload error:", error);
    }
  }

  // Wait for async tool data before rendering the form. Formik captures
  // initialValues on mount — if tools haven't loaded yet, the initial values
  // won't include MCP tool fields. Later, toggling those fields would make
  // the form permanently dirty since they have no baseline to compare against.
  if (isToolsLoading || isMcpLoading || isOpenApiLoading) {
    return null;
  }

  return (
    <>
      <div
        data-testid="AgentsEditorPage/container"
        aria-label="Agents Editor Page"
        className={cn(
          "h-full w-full",
          variant === "wizard" && "flex min-h-0 flex-col"
        )}
      >
        <Formik
          initialValues={initialValues}
          validationSchema={validationSchema}
          onSubmit={handleSubmit}
          validateOnChange
          validateOnBlur
          validateOnMount
          initialTouched={{
            description:
              initialValues.description.length >
              MAX_CHARACTERS_AGENT_DESCRIPTION,
            starter_messages: initialValues.starter_messages.map(
              (msg) => msg.length > MAX_CHARACTERS_STARTER_MESSAGE
            ) as unknown as boolean,
          }}
          initialStatus={{ warnings: {} }}
        >
          {({ isSubmitting, isValid, dirty, values, setFieldValue }) => {
            const fileStatusMap = new Map(
              allRecentFiles.map((f) => [f.id, f.status])
            );

            const hasUploadingFiles = values.user_file_ids.some(
              (fileId: string) => {
                const status = fileStatusMap.get(fileId);
                if (status === undefined) {
                  return fileId.startsWith("temp_");
                }
                return status === UserFileStatus.UPLOADING;
              }
            );

            const agentFormBody = (
              <AgentFormBody
                existingAgent={existingAgent}
                highlightedFields={highlightedFields}
                documentSets={documentSets}
                allRecentFiles={allRecentFiles}
                vectorDbEnabled={vectorDbEnabled}
                mcpServersWithTools={mcpServersWithTools}
                openApiTools={openApiTools}
                webSearchTool={webSearchTool}
                openURLTool={openURLTool}
                codeInterpreterTool={codeInterpreterTool}
                isImageGenerationAvailable={isImageGenerationAvailable}
                imageGenerationDisabledTooltip={
                  imageGenerationDisabledTooltip
                }
                getCurrentLlm={getCurrentLlm}
                onLlmSelect={onLlmSelect}
                llmProviders={llmProviders}
                canUpdateFeaturedStatus={canUpdateFeaturedStatus}
                onShareClick={() => shareAgentModal.toggle(true)}
                onDeleteClick={() => deleteAgentModal.toggle(true)}
                handleFileClick={handleFileClick}
                handleUploadChange={handleUploadChange}
              />
            );

            const agentPageHeader = (
              <SettingsLayouts.Header
                icon={SvgOnyxOctagon}
                dense={variant === "wizard"}
                title={existingAgent ? "Edit Agent" : "Create Agent"}
                rightChildren={
                  <div className="flex gap-2">
                    <OpalButton
                      prominence="secondary"
                      type="button"
                      onClick={() => router.back()}
                    >
                      Cancel
                    </OpalButton>
                    <SimpleTooltip
                      tooltip={
                        isSubmitting
                          ? "Saving changes..."
                          : !isValid
                            ? "Please fix the errors in the form before saving."
                            : !dirty
                              ? "No changes have been made."
                              : hasUploadingFiles
                                ? "Please wait for files to finish uploading."
                                : undefined
                      }
                      side="bottom"
                    >
                      <OpalButton
                        disabled={
                          isSubmitting ||
                          !isValid ||
                          !dirty ||
                          hasUploadingFiles
                        }
                        type="submit"
                      >
                        {existingAgent ? "Save" : "Create"}
                      </OpalButton>
                    </SimpleTooltip>
                  </div>
                }
                backButton
                separator
              />
            );

            return (
              <>
                <userFilesModal.Provider>
                  <UserFilesModal
                    title="User Files"
                    description="All files selected for this agent"
                    recentFiles={values.user_file_ids
                      .map((userFileId: string) => {
                        const rf = allRecentFiles.find(
                          (f) => f.id === userFileId
                        );
                        if (rf) return rf;
                        return {
                          id: userFileId,
                          name: `File ${userFileId.slice(0, 8)}`,
                          status: UserFileStatus.COMPLETED,
                          file_id: userFileId,
                          created_at: new Date().toISOString(),
                          project_id: null,
                          user_id: null,
                          file_type: "",
                          last_accessed_at: new Date().toISOString(),
                          chat_file_type: "file" as const,
                        } as unknown as ProjectFile;
                      })
                      .filter((f): f is ProjectFile => f !== null)}
                    selectedFileIds={values.user_file_ids}
                    onPickRecent={(file: ProjectFile) => {
                      if (!values.user_file_ids.includes(file.id)) {
                        setFieldValue("user_file_ids", [
                          ...values.user_file_ids,
                          file.id,
                        ]);
                      }
                    }}
                    onUnpickRecent={(file: ProjectFile) => {
                      setFieldValue(
                        "user_file_ids",
                        values.user_file_ids.filter((id) => id !== file.id)
                      );
                    }}
                    onView={(file: ProjectFile) => {
                      setPresentingDocument({
                        document_id: `project_file__${file.file_id}`,
                        semantic_identifier: file.name,
                      });
                    }}
                  />
                </userFilesModal.Provider>

                <shareAgentModal.Provider>
                  <ShareAgentModal
                    agentId={existingAgent?.id}
                    userIds={values.shared_user_ids}
                    groupIds={values.shared_group_ids}
                    isPublic={values.is_public}
                    isFeatured={values.is_featured}
                    labelIds={values.label_ids}
                    onShare={async (
                      userIds,
                      groupIds,
                      isPublic,
                      isFeatured,
                      labelIds
                    ) => {
                      if (!existingAgent) {
                        // New agents are not persisted until the main Create action.
                        setFieldValue("shared_user_ids", userIds);
                        setFieldValue("shared_group_ids", groupIds);
                        setFieldValue("is_public", isPublic);
                        setFieldValue("is_featured", isFeatured);
                        setFieldValue("label_ids", labelIds);
                        shareAgentModal.toggle(false);
                        return;
                      }

                      const applySharingFields = () => {
                        setFieldValue("shared_user_ids", userIds);
                        setFieldValue("shared_group_ids", groupIds);
                        setFieldValue("is_public", isPublic);
                        setFieldValue("label_ids", labelIds);
                      };

                      const refreshSharedUi = async () => {
                        try {
                          await refreshAgents();
                          refreshAgent?.();
                        } catch (error) {
                          console.error(
                            "Refresh failed after successful share:",
                            error
                          );
                          toast.error(
                            "Agent sharing was saved, but failed to refresh. Please reload."
                          );
                        }
                      };

                      let shareError: string | null;
                      try {
                        shareError = await updateAgentSharedStatus(
                          existingAgent.id,
                          userIds,
                          groupIds,
                          isPublic,
                          isPaidEnterpriseFeaturesEnabled,
                          labelIds
                        );
                      } catch (error) {
                        console.error(
                          "Share agent mutation failed unexpectedly:",
                          error
                        );
                        toast.error("Failed to share agent. Please try again.");
                        return;
                      }

                      if (shareError) {
                        toast.error(`Failed to share agent: ${shareError}`);
                        return;
                      }

                      if (canUpdateFeaturedStatus) {
                        let featuredError: string | null;
                        try {
                          featuredError = await updateAgentFeaturedStatus(
                            existingAgent.id,
                            isFeatured
                          );
                        } catch (error) {
                          console.error(
                            "Featured mutation failed unexpectedly:",
                            error
                          );
                          // Share succeeded; sync form and UI before returning.
                          applySharingFields();
                          await refreshSharedUi();
                          toast.error(
                            "Failed to update featured status. Please try again."
                          );
                          return;
                        }

                        if (featuredError) {
                          // Share succeeded, featured failed: keep modal open for retry.
                          applySharingFields();
                          await refreshSharedUi();
                          toast.error(
                            `Failed to update featured status: ${featuredError}`
                          );
                          return;
                        }

                        applySharingFields();
                        setFieldValue("is_featured", isFeatured);
                        shareAgentModal.toggle(false);
                        await refreshSharedUi();
                        return;
                      }

                      applySharingFields();
                      shareAgentModal.toggle(false);
                      await refreshSharedUi();
                    }}
                  />
                </shareAgentModal.Provider>
                <deleteAgentModal.Provider>
                  {deleteAgentModal.isOpen && (
                    <ConfirmationModalLayout
                      icon={SvgTrash}
                      title="Delete Agent"
                      submit={
                        <OpalButton
                          variant="danger"
                          onClick={handleDeleteAgent}
                        >
                          Delete Agent
                        </OpalButton>
                      }
                      onClose={() => deleteAgentModal.toggle(false)}
                    >
                      <GeneralLayouts.Section alignItems="start" gap={0.5}>
                        <Text>
                          Anyone using this agent will no longer be able to
                          access it. Deletion cannot be undone.
                        </Text>
                        <Text>Are you sure you want to delete this agent?</Text>
                      </GeneralLayouts.Section>
                    </ConfirmationModalLayout>
                  )}
                </deleteAgentModal.Provider>

                <div
                  className={cn(
                    variant === "wizard"
                      ? "flex min-h-0 flex-1 flex-col"
                      : "contents"
                  )}
                >
                  <Form
                    className={cn(
                      "min-h-0 w-full",
                      variant === "wizard"
                        ? "flex min-h-0 flex-1 flex-col"
                        : "h-full"
                    )}
                  >
                    <SettingsLayouts.Root
                      width={variant === "wizard" ? "full" : "md"}
                      scrollable={variant !== "wizard"}
                    >
                    {variant === "wizard" ? (
                      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
                        {agentPageHeader}
                        <div className="flex min-h-0 w-full min-w-0 flex-1 flex-row overflow-hidden">
                          <div className="flex min-h-0 w-[40%] min-w-0 flex-col overflow-hidden border-r border-border-02 bg-background-neutral-01">
                            <AgentBuilderChat
                              markFieldUpdated={markFieldUpdated}
                            />
                          </div>
                          <div className="min-h-0 min-w-0 flex-1 overflow-y-auto">
                            <SettingsLayouts.Body compact={variant === "wizard"}>
                              {agentFormBody}
                            </SettingsLayouts.Body>
                          </div>
                        </div>
                      </div>
                    ) : (
                      <>
                        {agentPageHeader}
                        <SettingsLayouts.Body>
                          {agentFormBody}
                        </SettingsLayouts.Body>
                      </>
                    )}
                    </SettingsLayouts.Root>
                  </Form>
                </div>
              </>
            );
          }}
        </Formik>
      </div>
    </>
  );
}
