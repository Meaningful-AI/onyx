import { LLMProviderName, LLMProviderView } from "@/interfaces/llm";
import AnthropicModal from "@/sections/modals/llmConfig/AnthropicModal";
import OpenAIModal from "@/sections/modals/llmConfig/OpenAIModal";
import OllamaModal from "@/sections/modals/llmConfig/OllamaModal";
import AzureModal from "@/sections/modals/llmConfig/AzureModal";
import VertexAIModal from "@/sections/modals/llmConfig/VertexAIModal";
import OpenRouterModal from "@/sections/modals/llmConfig/OpenRouterModal";
import CustomModal from "@/sections/modals/llmConfig/CustomModal";
import BedrockModal from "@/sections/modals/llmConfig/BedrockModal";
import LMStudioForm from "@/sections/modals/llmConfig/LMStudioForm";
import LiteLLMProxyModal from "@/sections/modals/llmConfig/LiteLLMProxyModal";
import BifrostModal from "@/sections/modals/llmConfig/BifrostModal";

export function getModalForExistingProvider(
  provider: LLMProviderView,
  onOpenChange?: (open: boolean) => void,
  defaultModelName?: string
) {
  const props = {
    existingLlmProvider: provider,
    onOpenChange,
    defaultModelName,
  };

  if (provider.is_custom_provider) {
    return <CustomModal {...props} />;
  }

  switch (provider.provider) {
    case LLMProviderName.OPENAI:
      return <OpenAIModal {...props} />;
    case LLMProviderName.ANTHROPIC:
      return <AnthropicModal {...props} />;
    case LLMProviderName.OLLAMA_CHAT:
      return <OllamaModal {...props} />;
    case LLMProviderName.AZURE:
      return <AzureModal {...props} />;
    case LLMProviderName.VERTEX_AI:
      return <VertexAIModal {...props} />;
    case LLMProviderName.BEDROCK:
      return <BedrockModal {...props} />;
    case LLMProviderName.OPENROUTER:
      return <OpenRouterModal {...props} />;
    case LLMProviderName.LM_STUDIO:
      return <LMStudioForm {...props} />;
    case LLMProviderName.LITELLM_PROXY:
      return <LiteLLMProxyModal {...props} />;
    case LLMProviderName.BIFROST:
      return <BifrostModal {...props} />;
    default:
      return <CustomModal {...props} />;
  }
}
