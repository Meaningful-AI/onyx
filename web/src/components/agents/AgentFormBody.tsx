"use client";

import { useEffect, useRef, useState } from "react";
import { FieldArray } from "formik";
import { useFormikContext } from "formik";
import * as GeneralLayouts from "@/layouts/general-layouts";
import * as InputLayouts from "@/layouts/input-layouts";
import * as ActionsLayouts from "@/layouts/actions-layouts";
import * as ExpandableCard from "@/layouts/expandable-card-layouts";
import InputTypeInField from "@/refresh-components/form/InputTypeInField";
import InputTextAreaField from "@/refresh-components/form/InputTextAreaField";
import InputTypeInElementField from "@/refresh-components/form/InputTypeInElementField";
import InputDatePickerField from "@/refresh-components/form/InputDatePickerField";
import Message from "@/refresh-components/messages/Message";
import Separator from "@/refresh-components/Separator";
import LLMSelector from "@/components/llm/LLMSelector";
import { STARTER_MESSAGES_EXAMPLES } from "@/lib/constants";
import Text from "@/refresh-components/texts/Text";
import { Card } from "@/refresh-components/cards";
import SimpleCollapsible from "@/refresh-components/SimpleCollapsible";
import SwitchField from "@/refresh-components/form/SwitchField";
import SimpleTooltip from "@/refresh-components/SimpleTooltip";
import Popover, { PopoverMenu } from "@/refresh-components/Popover";
import LineItem from "@/refresh-components/buttons/LineItem";
import {
  SvgActions,
  SvgExpand,
  SvgFold,
  SvgImage,
  SvgLock,
  SvgSliders,
  SvgUsers,
} from "@opal/icons";
import { Button as OpalButton } from "@opal/components";
import CustomAgentAvatar, {
  agentAvatarIconMap,
} from "@/refresh-components/avatars/CustomAgentAvatar";
import InputAvatar from "@/refresh-components/inputs/InputAvatar";
import SquareButton from "@/refresh-components/buttons/SquareButton";
import Button from "@/refresh-components/buttons/Button";
import { Hoverable } from "@opal/core";
import { buildImgUrl } from "@/app/app/components/files/images/utils";
import { FullPersona } from "@/app/admin/agents/interfaces";
import InputTypeIn from "@/refresh-components/inputs/InputTypeIn";
import useFilter from "@/hooks/useFilter";
import EnabledCount from "@/refresh-components/EnabledCount";
import { getActionIcon } from "@/lib/tools/mcpUtils";
import { MCPServer, MCPTool, ToolSnapshot } from "@/lib/tools/interfaces";
import { ProjectFile, UserFileStatus } from "@/app/app/projects/projectsService";
import AgentKnowledgePane from "@/sections/knowledge/AgentKnowledgePane";
import type { DocumentSetSummary } from "@/lib/types";
import SimpleLoader from "@/refresh-components/loaders/SimpleLoader";
import { cn } from "@/lib/utils";
interface AgentIconEditorProps {
  existingAgent?: FullPersona | null;
}

function FormWarningsEffect() {
  const { values, setStatus } = useFormikContext<{
    web_search: boolean;
    open_url: boolean;
  }>();

  useEffect(() => {
    const warnings: Record<string, string> = {};
    if (values.web_search && !values.open_url) {
      warnings.open_url =
        "Web Search without the ability to open URLs can lead to significantly worse web based results.";
    }
    setStatus({ warnings });
  }, [values.web_search, values.open_url, setStatus]);

  return null;
}

function AgentIconEditor({ existingAgent }: AgentIconEditorProps) {
  const { values, setFieldValue } = useFormikContext<{
    name: string;
    icon_name: string | null;
    uploaded_image_id: string | null;
    remove_image: boolean | null;
  }>();
  const [uploadedImagePreview, setUploadedImagePreview] = useState<
    string | null
  >(null);
  const [popoverOpen, setPopoverOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  async function handleImageUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploadedImagePreview(null);

    setFieldValue("icon_name", null);
    setFieldValue("remove_image", false);

    const reader = new FileReader();
    reader.onloadend = () => {
      setUploadedImagePreview(reader.result as string);
    };
    reader.readAsDataURL(file);

    try {
      const formData = new FormData();
      formData.append("file", file);
      const response = await fetch("/api/admin/persona/upload-image", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        console.error("Failed to upload image");
        setUploadedImagePreview(null);
        return;
      }

      const { file_id } = await response.json();
      setFieldValue("uploaded_image_id", file_id);
      setPopoverOpen(false);
    } catch (error) {
      console.error("Upload error:", error);
      setUploadedImagePreview(null);
    }
  }

  const imageSrc = uploadedImagePreview
    ? uploadedImagePreview
    : values.uploaded_image_id
      ? buildImgUrl(values.uploaded_image_id)
      : values.icon_name
        ? undefined
        : values.remove_image
          ? undefined
          : existingAgent?.uploaded_image_id
            ? buildImgUrl(existingAgent.uploaded_image_id)
            : undefined;

  function handleIconClick(iconName: string | null) {
    setFieldValue("icon_name", iconName);
    setFieldValue("uploaded_image_id", null);
    setFieldValue("remove_image", true);
    setUploadedImagePreview(null);
    setPopoverOpen(false);

    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  }

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        onChange={handleImageUpload}
        className="hidden"
      />

      <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
        <Popover.Trigger asChild>
          <Hoverable.Root group="inputAvatar" widthVariant="fit">
            <InputAvatar className="relative flex flex-col items-center justify-center h-[7.5rem] w-[7.5rem]">
              <CustomAgentAvatar
                size={imageSrc ? 7.5 * 16 : 40}
                src={imageSrc}
                iconName={values.icon_name ?? undefined}
                name={values.name}
              />
              <div className="absolute bottom-0 left-1/2 -translate-x-1/2 mb-2">
                <Hoverable.Item group="inputAvatar" variant="opacity-on-hover">
                  <Button className="h-[1.75rem]" secondary>
                    Edit
                  </Button>
                </Hoverable.Item>
              </div>
            </InputAvatar>
          </Hoverable.Root>
        </Popover.Trigger>
        <Popover.Content>
          <PopoverMenu>
            {[
              <LineItem
                key="upload-image"
                icon={SvgImage}
                onClick={() => fileInputRef.current?.click()}
                emphasized
              >
                Upload Image
              </LineItem>,
              null,
              <div className="grid grid-cols-4 gap-1">
                <SquareButton
                  key="default-icon"
                  icon={() => (
                    <CustomAgentAvatar name={values.name} size={30} />
                  )}
                  onClick={() => handleIconClick(null)}
                  transient={!imageSrc && values.icon_name === null}
                />
                {Object.keys(agentAvatarIconMap).map((iconName) => (
                  <SquareButton
                    key={iconName}
                    onClick={() => handleIconClick(iconName)}
                    icon={() => (
                      <CustomAgentAvatar iconName={iconName} size={30} />
                    )}
                    transient={values.icon_name === iconName}
                  />
                ))}
              </div>,
            ]}
          </PopoverMenu>
        </Popover.Content>
      </Popover>
    </>
  );
}

interface OpenApiToolCardProps {
  tool: ToolSnapshot;
}

function OpenApiToolCard({ tool }: OpenApiToolCardProps) {
  const toolFieldName = `openapi_tool_${tool.id}`;

  return (
    <ExpandableCard.Root defaultFolded>
      <ActionsLayouts.Header
        title={tool.display_name || tool.name}
        description={tool.description}
        icon={SvgActions}
        rightChildren={<SwitchField name={toolFieldName} />}
      />
    </ExpandableCard.Root>
  );
}

interface MCPServerCardProps {
  server: MCPServer;
  tools: MCPTool[];
  isLoading: boolean;
}

function MCPServerCard({
  server,
  tools: enabledTools,
  isLoading,
}: MCPServerCardProps) {
  const [isFolded, setIsFolded] = useState(false);
  const { values, setFieldValue, getFieldMeta } = useFormikContext<any>();
  const serverFieldName = `mcp_server_${server.id}`;
  const isServerEnabled = values[serverFieldName]?.enabled ?? false;
  const {
    query,
    setQuery,
    filtered: filteredTools,
  } = useFilter(enabledTools, (tool) => `${tool.name} ${tool.description}`);

  const enabledCount = enabledTools.filter((tool) => {
    const toolFieldValue = values[serverFieldName]?.[`tool_${tool.id}`];
    return toolFieldValue === true;
  }).length;

  return (
    <ExpandableCard.Root isFolded={isFolded} onFoldedChange={setIsFolded}>
      <ActionsLayouts.Header
        title={server.name}
        description={server.description}
        icon={getActionIcon(server.server_url, server.name)}
        rightChildren={
          <GeneralLayouts.Section
            flexDirection="row"
            gap={0.5}
            alignItems="start"
          >
            <EnabledCount
              enabledCount={enabledCount}
              totalCount={enabledTools.length}
            />
            <SwitchField
              name={`${serverFieldName}.enabled`}
              onCheckedChange={(checked) => {
                enabledTools.forEach((tool) => {
                  setFieldValue(`${serverFieldName}.tool_${tool.id}`, checked);
                });
                if (!checked) return;
                setIsFolded(false);
              }}
            />
          </GeneralLayouts.Section>
        }
      >
        <GeneralLayouts.Section flexDirection="row" gap={0.5}>
          <InputTypeIn
            placeholder="Search tools..."
            variant="internal"
            leftSearchIcon
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {enabledTools.length > 0 && (
            <OpalButton
              prominence="internal"
              rightIcon={isFolded ? SvgExpand : SvgFold}
              onClick={() => setIsFolded((prev) => !prev)}
            >
              {isFolded ? "Expand" : "Fold"}
            </OpalButton>
          )}
        </GeneralLayouts.Section>
      </ActionsLayouts.Header>
      {isLoading ? (
        <ActionsLayouts.Content>
          <GeneralLayouts.Section padding={1}>
            <SimpleLoader />
          </GeneralLayouts.Section>
        </ActionsLayouts.Content>
      ) : (
        enabledTools.length > 0 &&
        filteredTools.length > 0 && (
          <ActionsLayouts.Content>
            {filteredTools.map((tool) => (
              <ActionsLayouts.Tool
                key={tool.id}
                name={`${serverFieldName}.tool_${tool.id}`}
                title={tool.name}
                description={tool.description}
                icon={tool.icon ?? SvgSliders}
                disabled={
                  !tool.isAvailable ||
                  !getFieldMeta<boolean>(`${serverFieldName}.enabled`).value
                }
                rightChildren={
                  <SwitchField
                    name={`${serverFieldName}.tool_${tool.id}`}
                    disabled={!isServerEnabled}
                  />
                }
              />
            ))}
          </ActionsLayouts.Content>
        )
      )}
    </ExpandableCard.Root>
  );
}

function StarterMessages() {
  const max_starters = STARTER_MESSAGES_EXAMPLES.length;

  const { values } = useFormikContext<{
    starter_messages: string[];
  }>();

  const starters = values.starter_messages || [];

  const filledStarters = starters.filter((s) => s).length;
  const canAddMore = filledStarters < max_starters;

  const visibleCount = Math.min(
    max_starters,
    Math.max(
      1,
      filledStarters === 0 ? 1 : filledStarters + (canAddMore ? 1 : 0)
    )
  );

  return (
    <FieldArray name="starter_messages">
      {(arrayHelpers) => (
        <GeneralLayouts.Section gap={0.5}>
          {Array.from({ length: visibleCount }, (_, i) => (
            <InputTypeInElementField
              key={`starter_messages.${i}`}
              name={`starter_messages.${i}`}
              placeholder={
                STARTER_MESSAGES_EXAMPLES[i] ||
                "Enter a conversation starter..."
              }
              onRemove={() => arrayHelpers.remove(i)}
            />
          ))}
        </GeneralLayouts.Section>
      )}
    </FieldArray>
  );
}

export interface AgentFormBodyProps {
  existingAgent?: FullPersona | null;
  highlightedFields?: Set<string>;
  documentSets: DocumentSetSummary[] | undefined;
  allRecentFiles: ProjectFile[];
  vectorDbEnabled: boolean;
  mcpServersWithTools: Array<{
    server: MCPServer;
    tools: MCPTool[];
    isLoading: boolean;
  }>;
  openApiTools: ToolSnapshot[];
  webSearchTool: ToolSnapshot | undefined;
  openURLTool: ToolSnapshot | undefined;
  codeInterpreterTool: ToolSnapshot | undefined;
  isImageGenerationAvailable: boolean;
  imageGenerationDisabledTooltip: string | undefined;
  getCurrentLlm: (values: any, llmProviders: any) => string | null;
  onLlmSelect: (selected: string | null, setFieldValue: any) => void;
  llmProviders: any[] | undefined;
  canUpdateFeaturedStatus: boolean;
  onShareClick: () => void;
  onDeleteClick: () => void;
  handleFileClick: (file: ProjectFile) => void;
  handleUploadChange: (
    e: React.ChangeEvent<HTMLInputElement>,
    currentFileIds: string[],
    setFieldValue: (field: string, value: unknown) => void
  ) => void;
}

function FieldHighlight({
  fieldKey,
  highlightedFields,
  children,
}: {
  fieldKey: string;
  highlightedFields?: Set<string>;
  children: React.ReactNode;
}) {
  const on = highlightedFields?.has(fieldKey);
  return (
    <div
      className={cn(
        "w-full rounded-md",
        on && "ring-2 ring-accent transition-all duration-500"
      )}
    >
      {children}
    </div>
  );
}

export default function AgentFormBody({
  existingAgent,
  highlightedFields,
  documentSets,
  allRecentFiles,
  vectorDbEnabled,
  mcpServersWithTools,
  openApiTools,
  webSearchTool,
  openURLTool,
  codeInterpreterTool,
  isImageGenerationAvailable,
  imageGenerationDisabledTooltip,
  getCurrentLlm,
  onLlmSelect,
  llmProviders,
  canUpdateFeaturedStatus,
  onShareClick,
  onDeleteClick,
  handleFileClick,
  handleUploadChange,
}: AgentFormBodyProps) {
  const { values, setFieldValue } = useFormikContext<any>();

  const fileStatusMap = new Map(
    allRecentFiles.map((f) => [f.id, f.status])
  );

  const hasProcessingFiles = values.user_file_ids.some(
    (fileId: string) =>
      fileStatusMap.get(fileId) === UserFileStatus.PROCESSING
  );

  const isShared =
    values.is_public ||
    values.shared_user_ids.length > 0 ||
    values.shared_group_ids.length > 0;

  return (
    <>
      <FormWarningsEffect />

      <GeneralLayouts.Section
        flexDirection="row"
        gap={2.5}
        alignItems="start"
        justifyContent="start"
        width="full"
      >
        <GeneralLayouts.Section
          justifyContent="start"
          alignItems="stretch"
          width="full"
          className="min-w-0 flex-1"
        >
          <FieldHighlight fieldKey="name" highlightedFields={highlightedFields}>
            <InputLayouts.Vertical name="name" title="Name">
              <InputTypeInField
                name="name"
                placeholder="Name your agent"
              />
            </InputLayouts.Vertical>
          </FieldHighlight>

          <FieldHighlight
            fieldKey="description"
            highlightedFields={highlightedFields}
          >
            <InputLayouts.Vertical
              name="description"
              title="Description"
              suffix="optional"
            >
              <InputTextAreaField
                name="description"
                placeholder="What does this agent do?"
              />
            </InputLayouts.Vertical>
          </FieldHighlight>
        </GeneralLayouts.Section>

        <GeneralLayouts.Section
          width="fit"
          alignItems="start"
          justifyContent="start"
          className="shrink-0"
        >
          <InputLayouts.Vertical name="agent_avatar" title="Agent Avatar">
            <AgentIconEditor existingAgent={existingAgent} />
          </InputLayouts.Vertical>
        </GeneralLayouts.Section>
      </GeneralLayouts.Section>

      <Separator noPadding />

      <GeneralLayouts.Section
        justifyContent="start"
        alignItems="stretch"
        width="full"
      >
        <FieldHighlight
          fieldKey="instructions"
          highlightedFields={highlightedFields}
        >
          <InputLayouts.Vertical
            name="instructions"
            title="Instructions"
            suffix="optional"
            description="Add instructions to tailor the response for this agent."
          >
            <InputTextAreaField
              name="instructions"
              placeholder="Think step by step and show reasoning for complex problems. Use specific examples. Emphasize action items, and leave blanks for the human to fill in when you have unknown. Use a polite enthusiastic tone."
            />
          </InputLayouts.Vertical>
        </FieldHighlight>

        <FieldHighlight
          fieldKey="starter_messages"
          highlightedFields={highlightedFields}
        >
          <InputLayouts.Vertical
            name="starter_messages"
            title="Conversation Starters"
            description="Example messages that help users understand what this agent can do and how to interact with it effectively."
            suffix="optional"
          >
            <StarterMessages />
          </InputLayouts.Vertical>
        </FieldHighlight>
      </GeneralLayouts.Section>

      <Separator noPadding />

      <AgentKnowledgePane
        enableKnowledge={values.enable_knowledge}
        onEnableKnowledgeChange={(enabled) =>
          setFieldValue("enable_knowledge", enabled)
        }
        selectedSources={values.selected_sources}
        onSourcesChange={(sources) =>
          setFieldValue("selected_sources", sources)
        }
        documentSets={documentSets ?? []}
        selectedDocumentSetIds={values.document_set_ids}
        onDocumentSetIdsChange={(ids) =>
          setFieldValue("document_set_ids", ids)
        }
        selectedDocumentIds={values.document_ids}
        onDocumentIdsChange={(ids) => setFieldValue("document_ids", ids)}
        selectedFolderIds={values.hierarchy_node_ids}
        onFolderIdsChange={(ids) =>
          setFieldValue("hierarchy_node_ids", ids)
        }
        selectedFileIds={values.user_file_ids}
        onFileIdsChange={(ids) => setFieldValue("user_file_ids", ids)}
        allRecentFiles={allRecentFiles}
        onFileClick={handleFileClick}
        onUploadChange={(e) =>
          handleUploadChange(e, values.user_file_ids, setFieldValue)
        }
        hasProcessingFiles={hasProcessingFiles}
        initialAttachedDocuments={existingAgent?.attached_documents}
        initialHierarchyNodes={existingAgent?.hierarchy_nodes}
        vectorDbEnabled={vectorDbEnabled}
      />

      <Separator noPadding />

      <SimpleCollapsible>
        <SimpleCollapsible.Header
          title="Actions"
          description="Tools and capabilities available for this agent to use."
        />
        <SimpleCollapsible.Content>
          <GeneralLayouts.Section
            gap={0.5}
            justifyContent="start"
            alignItems="stretch"
            width="full"
          >
            <FieldHighlight
              fieldKey="image_generation"
              highlightedFields={highlightedFields}
            >
              <SimpleTooltip
                tooltip={imageGenerationDisabledTooltip}
                side="top"
              >
                <Card
                  variant={
                    isImageGenerationAvailable ? undefined : "disabled"
                  }
                >
                  <InputLayouts.Horizontal
                    name="image_generation"
                    title="Image Generation"
                    description="Generate and manipulate images using AI-powered tools."
                    disabled={!isImageGenerationAvailable}
                  >
                    <SwitchField
                      name="image_generation"
                      disabled={!isImageGenerationAvailable}
                    />
                  </InputLayouts.Horizontal>
                </Card>
              </SimpleTooltip>
            </FieldHighlight>

            <FieldHighlight
              fieldKey="web_search"
              highlightedFields={highlightedFields}
            >
              <Card variant={!!webSearchTool ? undefined : "disabled"}>
                <InputLayouts.Horizontal
                  name="web_search"
                  title="Web Search"
                  description="Search the web for real-time information and up-to-date results."
                  disabled={!webSearchTool}
                >
                  <SwitchField
                    name="web_search"
                    disabled={!webSearchTool}
                  />
                </InputLayouts.Horizontal>
              </Card>
            </FieldHighlight>

            <Card variant={!!openURLTool ? undefined : "disabled"}>
              <InputLayouts.Horizontal
                name="open_url"
                title="Open URL"
                description="Fetch and read content from web URLs."
                disabled={!openURLTool}
              >
                <SwitchField name="open_url" disabled={!openURLTool} />
              </InputLayouts.Horizontal>
            </Card>

            <FieldHighlight
              fieldKey="code_interpreter"
              highlightedFields={highlightedFields}
            >
              <Card
                variant={!!codeInterpreterTool ? undefined : "disabled"}
              >
                <InputLayouts.Horizontal
                  name="code_interpreter"
                  title="Code Interpreter"
                  description="Generate and run code."
                  disabled={!codeInterpreterTool}
                >
                  <SwitchField
                    name="code_interpreter"
                    disabled={!codeInterpreterTool}
                  />
                </InputLayouts.Horizontal>
              </Card>
            </FieldHighlight>

            <>
              {(mcpServersWithTools.length > 0 || openApiTools.length > 0) && (
                <Separator noPadding className="py-1" />
              )}

              {mcpServersWithTools.length > 0 && (
                <GeneralLayouts.Section
                  gap={0.5}
                  justifyContent="start"
                  alignItems="stretch"
                  width="full"
                >
                  {mcpServersWithTools.map(
                    ({ server, tools, isLoading }) => (
                      <MCPServerCard
                        key={server.id}
                        server={server}
                        tools={tools}
                        isLoading={isLoading}
                      />
                    )
                  )}
                </GeneralLayouts.Section>
              )}

              {openApiTools.length > 0 && (
                <GeneralLayouts.Section
                  gap={0.5}
                  justifyContent="start"
                  alignItems="stretch"
                  width="full"
                >
                  {openApiTools.map((tool) => (
                    <OpenApiToolCard key={tool.id} tool={tool} />
                  ))}
                </GeneralLayouts.Section>
              )}
            </>
          </GeneralLayouts.Section>
        </SimpleCollapsible.Content>
      </SimpleCollapsible>

      <Separator noPadding />

      <SimpleCollapsible>
        <SimpleCollapsible.Header
          title="Advanced Options"
          description="Fine-tune agent prompts and knowledge."
        />
        <SimpleCollapsible.Content>
          <GeneralLayouts.Section
            justifyContent="start"
            alignItems="stretch"
            width="full"
          >
            <Card>
              <InputLayouts.Horizontal
                title="Share This Agent"
                description="with other users, groups, or everyone in your organization."
                center
              >
                <OpalButton
                  prominence="secondary"
                  icon={isShared ? SvgUsers : SvgLock}
                  onClick={onShareClick}
                >
                  Share
                </OpalButton>
              </InputLayouts.Horizontal>
              {canUpdateFeaturedStatus && (
                <>
                  <InputLayouts.Horizontal
                    name="is_featured"
                    title="Feature This Agent"
                    description="Show this agent at the top of the explore agents list and automatically pin it to the sidebar for new users with access."
                  >
                    <SwitchField name="is_featured" />
                  </InputLayouts.Horizontal>
                  {values.is_featured && !isShared && (
                    <Message
                      static
                      close={false}
                      className="w-full"
                      text="This agent is private to you and will only be featured for yourself."
                    />
                  )}
                </>
              )}
            </Card>

            <Card>
              <InputLayouts.Horizontal
                name="llm_model"
                title="Default Model"
                description="This model will be used by Meaningful AI by default in your chats."
              >
                <LLMSelector
                  name="llm_model"
                  llmProviders={llmProviders ?? []}
                  currentLlm={getCurrentLlm(values, llmProviders)}
                  onSelect={(selected) =>
                    onLlmSelect(selected, setFieldValue)
                  }
                />
              </InputLayouts.Horizontal>
              <InputLayouts.Horizontal
                name="knowledge_cutoff_date"
                title="Knowledge Cutoff Date"
                suffix="optional"
                description="Documents with a last-updated date prior to this will be ignored."
              >
                <InputDatePickerField
                  name="knowledge_cutoff_date"
                  maxDate={new Date()}
                />
              </InputLayouts.Horizontal>
              <InputLayouts.Horizontal
                name="replace_base_system_prompt"
                title="Overwrite System Prompt"
                suffix="(Not Recommended)"
                description='Remove the base system prompt which includes useful instructions (e.g. "You can use Markdown tables"). This may affect response quality.'
              >
                <SwitchField name="replace_base_system_prompt" />
              </InputLayouts.Horizontal>
            </Card>

            <GeneralLayouts.Section
              gap={0.25}
              justifyContent="start"
              alignItems="stretch"
              width="full"
            >
              <InputLayouts.Vertical
                name="reminders"
                title="Reminders"
                suffix="optional"
              >
                <InputTextAreaField
                  name="reminders"
                  placeholder="Remember, I want you to always format your response as a numbered list."
                />
              </InputLayouts.Vertical>
              <Text text03 secondaryBody>
                Append a brief reminder to the prompt messages. Use this to
                remind the agent if you find that it tends to forget certain
                instructions as the chat progresses. This should be brief and
                not interfere with the user messages.
              </Text>
            </GeneralLayouts.Section>
          </GeneralLayouts.Section>
        </SimpleCollapsible.Content>
      </SimpleCollapsible>

      {existingAgent && (
        <>
          <Separator noPadding />

          <Card>
            <InputLayouts.Horizontal
              title="Delete This Agent"
              description="Anyone using this agent will no longer be able to access it."
              center
            >
              <OpalButton
                variant="danger"
                prominence="secondary"
                onClick={onDeleteClick}
              >
                Delete Agent
              </OpalButton>
            </InputLayouts.Horizontal>
          </Card>
        </>
      )}
    </>
  );
}
