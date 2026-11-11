"""The models a step can run on, and the steps that can carry a default model.

llm-service owns this list because it is the service that routes a model id to
a provider ("ollama" and "ollama/<name>" go to Ollama, anything else to Hive).
Other services read it over HTTP instead of keeping a second copy.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ModelOption:
    id: str
    label: str
    #: Served on this machine rather than by Hive; the GUI warns it can time out
    #: on the code step and on the drawings.
    local: bool = False

    def to_dict(self) -> dict:
        return {"id": self.id, "label": self.label, "local": self.local}


#: The first model is the default of every step with none saved.
CATALOG: tuple[ModelOption, ...] = (
    ModelOption("ollama", "Ollama (local AI)", local=True),
    ModelOption("deepseek-ai/deepseek-v4.1-flash", "DeepSeek V4.1 Flash"),
    ModelOption("zai-org/glm-5.3-flash", "GLM-5.3-Flash"),
)

#: Steps that call an AI and can carry a default model: the seven of a video's
#: own model choices in authoring-service, the publish step, the drawings of the
#: illustration library, which belong to no video, and visual-service's visual
#: concept step.
STEPS: tuple[str, ...] = (
    "brief", "research", "story_frame", "beat_sheet", "story", "storyboard", "code", "publish", "illustrations",
    "visual_concept", "visual_plan",
)


def valid_model(model_id: str) -> bool:
    """True for "" (the server's configured default) or a model of the catalog."""
    return model_id == "" or any(m.id == model_id for m in CATALOG)


def valid_step(step: str) -> bool:
    return step in STEPS


def with_fallback(saved: dict[str, str]) -> dict[str, str]:
    """The default model of every step: the one saved for it, else the catalog's first."""
    return {step: saved.get(step) or CATALOG[0].id for step in STEPS}
