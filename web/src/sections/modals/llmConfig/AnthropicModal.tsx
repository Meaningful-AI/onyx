"use client";

import { useState } from "react";
import { useSWRConfig } from "swr";
import { Formik } from "formik";
import { LLMProviderFormProps } from "@/interfaces/llm";
import * as Yup from "yup";
import { useWellKnownLLMProvider } from "@/hooks/useLLMProviders";
import {
  buildDefaultInitialValues,
  buildDefaultValidationSchema,
  buildAvailableModelConfigurations,
  buildOnboardingInitialValues,
} from "@/sections/modals/llmConfig/utils";
import {
  submitLLMProvider,
  submitOnboardingProvider,
} from "@/sections/modals/llmConfig/svc";
import {
  APIKeyField,
  ModelsField,
  DisplayNameField,
  ModelsAccessField,
  FieldSeparator,
  SingleDefaultModelField,
  LLMConfigurationModalWrapper,
} from "@/sections/modals/llmConfig/shared";
import { setDefaultLlmModel } from "@/lib/llmConfig/svc";
import { refreshLlmProviderCaches } from "@/lib/llmConfig/cache";
import { toast } from "@/hooks/useToast";

const ANTHROPIC_PROVIDER_NAME = "anthropic";
const DEFAULT_DEFAULT_MODEL_NAME = "claude-sonnet-4-5";

export default function AnthropicModal({
  variant = "llm-configuration",
  existingLlmProvider,
  shouldMarkAsDefault,
  onOpenChange,
  globalDefault,
  onboardingState,
  onboardingActions,
  llmDescriptor,
}: LLMProviderFormProps) {
  const isOnboarding = variant === "onboarding";
  const [isTesting, setIsTesting] = useState(false);
  const [pendingDefault, setPendingDefault] = useState<string | null>(null);
  const { mutate } = useSWRConfig();
  const { wellKnownLLMProvider } = useWellKnownLLMProvider(
    ANTHROPIC_PROVIDER_NAME
  );

  const onClose = () => onOpenChange?.(false);

  const modelConfigurations = buildAvailableModelConfigurations(
    existingLlmProvider,
    wellKnownLLMProvider ?? llmDescriptor
  );

  const initialValues = isOnboarding
    ? {
        ...buildOnboardingInitialValues(),
        name: ANTHROPIC_PROVIDER_NAME,
        provider: ANTHROPIC_PROVIDER_NAME,
        api_key: "",
        default_model_name: DEFAULT_DEFAULT_MODEL_NAME,
      }
    : {
        ...buildDefaultInitialValues(existingLlmProvider, modelConfigurations),
        api_key: existingLlmProvider?.api_key ?? "",
        api_base: existingLlmProvider?.api_base ?? undefined,
        default_model_name:
          existingLlmProvider?.model_configurations?.[0]?.name ??
          wellKnownLLMProvider?.recommended_default_model?.name ??
          DEFAULT_DEFAULT_MODEL_NAME,
        is_auto_mode: existingLlmProvider?.is_auto_mode ?? true,
      };

  const validationSchema = isOnboarding
    ? Yup.object().shape({
        api_key: Yup.string().required("API Key is required"),
        default_model_name: Yup.string().required("Model name is required"),
      })
    : buildDefaultValidationSchema().shape({
        api_key: Yup.string().required("API Key is required"),
      });

  return (
    <Formik
      initialValues={initialValues}
      validationSchema={validationSchema}
      validateOnMount={true}
      onSubmit={async (values, { setSubmitting }) => {
        if (isOnboarding && onboardingState && onboardingActions) {
          const modelConfigsToUse =
            (wellKnownLLMProvider ?? llmDescriptor)?.known_models ?? [];

          await submitOnboardingProvider({
            providerName: ANTHROPIC_PROVIDER_NAME,
            payload: {
              ...values,
              model_configurations: modelConfigsToUse,
              is_auto_mode:
                values.default_model_name === DEFAULT_DEFAULT_MODEL_NAME,
            },
            onboardingState,
            onboardingActions,
            isCustomProvider: false,
            onClose,
            setIsSubmitting: setSubmitting,
          });
        } else {
          await submitLLMProvider({
            providerName: ANTHROPIC_PROVIDER_NAME,
            values,
            initialValues,
            modelConfigurations,
            existingLlmProvider,
            pendingDefaultModelName:
              pendingDefault ??
              (shouldMarkAsDefault ? values.default_model_name : undefined),
            setIsTesting,
            mutate,
            onClose,
            setSubmitting,
          });
        }
      }}
    >
      {(formikProps) => (
        <LLMConfigurationModalWrapper
          providerEndpoint={ANTHROPIC_PROVIDER_NAME}
          existingProviderName={existingLlmProvider?.name}
          onClose={onClose}
          isFormValid={formikProps.isValid}
          isDirty={formikProps.dirty}
          isTesting={isTesting}
          isSubmitting={formikProps.isSubmitting}
        >
          <APIKeyField providerName="Anthropic" />

          {!isOnboarding && (
            <>
              <FieldSeparator />
              <DisplayNameField disabled={!!existingLlmProvider} />
            </>
          )}

          <FieldSeparator />
          {isOnboarding ? (
            <SingleDefaultModelField placeholder="E.g. claude-sonnet-4-5" />
          ) : (
            <ModelsField
              modelConfigurations={modelConfigurations}
              formikProps={formikProps}
              recommendedDefaultModel={
                wellKnownLLMProvider?.recommended_default_model ?? null
              }
              shouldShowAutoUpdateToggle={true}
              globalDefault={globalDefault}
              providerId={existingLlmProvider?.id}
              onSetGlobalDefault={
                existingLlmProvider
                  ? async (modelName) => {
                      try {
                        await setDefaultLlmModel(
                          existingLlmProvider.id,
                          modelName
                        );
                        await refreshLlmProviderCaches(mutate);
                        toast.success("Default model updated successfully!");
                      } catch (e) {
                        const msg =
                          e instanceof Error ? e.message : "Unknown error";
                        toast.error(`Failed to set default model: ${msg}`);
                      }
                    }
                  : (modelName) => setPendingDefault(modelName)
              }
            />
          )}

          {!isOnboarding && (
            <>
              <FieldSeparator />
              <ModelsAccessField formikProps={formikProps} />
            </>
          )}
        </LLMConfigurationModalWrapper>
      )}
    </Formik>
  );
}
