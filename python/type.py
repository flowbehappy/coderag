from typing import *
from dataclasses import dataclass, field


@dataclass
class ReasoningText:
    tag: str = "reasoning_text"
    text: str = ""


@dataclass
class MessagePostReasoningContent:
    tag: str = "reasoning_content"
    text: str = ""
    reasoningText: ReasoningText = field(default_factory=ReasoningText)
    style: List[str] = field(default_factory=list)


@dataclass
class MessagePostText:
    tag: str = "text"
    text: str = ""
    style: List[str] = field(default_factory=list)


@dataclass
class MessagePostLink:
    tag: str = "a"
    href: str = ""
    text: str = ""
    style: List[str] = field(default_factory=list)


@dataclass
class MessagePostAt:
    tag: str = "at"
    user_id: str = ""
    user_name: str = ""
    style: List[str] = field(default_factory=list)


@dataclass
class MessagePostImage:
    tag: str = "img"
    image_key: str = ""


@dataclass
class MessagePostMedia:
    tag: str = "media"
    file_key: str = ""
    image_key: str = ""


@dataclass
class MessagePostEmotion:
    tag: str = "emotion"
    emoji_type: str = ""


@dataclass
class MessagePostHr:
    tag: str = "hr"


@dataclass
class MessagePostCode:
    tag: str = "code_block"
    language: str = ""
    text: str = ""


MessagePostElement = Union[
    MessagePostReasoningContent,
    MessagePostText, MessagePostLink, MessagePostAt,
    MessagePostImage, MessagePostMedia, MessagePostEmotion,
    MessagePostHr, MessagePostCode
]


@dataclass
class MessagePostContent:
    title: str = ""
    content: List[List[MessagePostElement]] = field(default_factory=list)
    from_app: bool = False


@dataclass
class Summary:
    content: str = ""
    i18n_content: Dict[str, str] = field(default_factory=dict)


@dataclass
class CardConfig:
    streaming_mode: bool = True
    enable_forward: bool = False
    update_multi: bool = False
    width_mode: str = ""
    enable_forward_interaction: bool = False
    summary: Summary = field(default_factory=Summary)

    def set_streaming_mode(self, mode: bool):
        self.streaming_mode = mode

    def update_summary(self, summary: Summary):
        self.summary = summary
