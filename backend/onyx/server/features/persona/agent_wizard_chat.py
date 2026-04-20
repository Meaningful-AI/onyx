"""Streaming chat for the agent creation wizard (natural-language form fill)."""

from __future__ import annotations

import json
from collections.abc import Iterator
from typing import Any
from typing import Literal

from pydantic import BaseModel

from onyx.llm.factory import get_default_llm
from onyx.llm.interfaces import LLM
from onyx.llm.models import AssistantMessage
from onyx.llm.models import LanguageModelInput
from onyx.llm.models import SystemMessage
from onyx.llm.models import UserMessage
from onyx.utils.logger import setup_logger

logger = setup_logger()


AGENT_WIZARD_SYSTEM_PROMPT = """You are helping a user create an AI agent in Meaningful AI / Onyx.

Respond conversationally: confirm what you set, suggest sensible next steps, and ask short follow-ups when helpful.

You must infer agent settings from the conversation. After your conversational reply, output a machine-readable block with the current best guess for fields that should change.

Rules for the <<<FIELDS>>> block:
- Use markers exactly: <<<FIELDS>>> on its own line, then a single JSON object, then <<<END>>> on its own line.
- Only include JSON keys for fields that should change from the current form values (see Current form values below).
- Valid keys: "name", "description", "instructions", "starter_messages" (array of strings), "web_search", "image_generation", "code_interpreter" (booleans).
- "instructions" is the agent's system / behavior instructions (not the short marketing description).
- "starter_messages": up to 5 short example user messages; omit or use [] if none.
- Do not repeat unchanged values.

Current form values (JSON):
"""


class WizardChatMessage(BaseModel):
    role: Literal["user", "assistant"]
    content: str


class AgentWizardChatRequest(BaseModel):
    messages: list[WizardChatMessage]
    current_values: dict[str, Any]


def _build_language_model_input(
    body: AgentWizardChatRequest,
) -> LanguageModelInput:
    system_text = (
        AGENT_WIZARD_SYSTEM_PROMPT
        + json.dumps(body.current_values, ensure_ascii=False)
    )
    messages: list[SystemMessage | UserMessage | AssistantMessage] = [
        SystemMessage(content=system_text)
    ]
    for m in body.messages:
        if m.role == "user":
            messages.append(UserMessage(content=m.content))
        else:
            messages.append(AssistantMessage(content=m.content))
    return messages


def sse_delta_generator(
    body: AgentWizardChatRequest,
    llm: LLM,
) -> Iterator[str]:
    prompt = _build_language_model_input(body)
    try:
        for chunk in llm.stream(prompt):
            delta = chunk.choice.delta.content
            if delta:
                yield f"data: {json.dumps({'delta': delta})}\n\n"
    except Exception:
        logger.exception("Agent wizard streaming failed; falling back to non-streaming")
        try:
            response = llm.invoke(prompt)
            text = response.choice.message.content or ""
            yield f"data: {json.dumps({'delta': text})}\n\n"
        except Exception:
            logger.exception("Agent wizard non-streaming fallback failed")
            yield f"data: {json.dumps({'error': 'llm_failed'})}\n\n"


def get_wizard_llm() -> LLM:
    return get_default_llm()
