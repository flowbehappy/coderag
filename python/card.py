import json
from typing import Optional
import lark_oapi as lark
from lark_oapi.api.cardkit.v1 import *
from type import *

CARD_CONTEXT = ""
with open("card.json") as f:
    CARD_CONTEXT = f.read()


def create_card(client: lark.Client, card_content: str = CARD_CONTEXT) -> Optional[str]:
    req = CreateCardRequest.builder() \
        .request_body(CreateCardRequestBody.builder()
                      .type("card_json")
                      .data(card_content)
                      .build()) \
        .build()

    resp = client.cardkit.v1.card.create(req)
    if not resp.success():
        print(f"Error creating card: {resp.code}, {resp.msg}")
        return None

    return resp.data.card_id


def update_card_config(client: lark.Client, card_id: str, config: CardConfig) -> bool:
    try:
        config_data = json.dumps({
            "streaming_mode": config.streaming_mode,
            "enable_forward": config.enable_forward,
            "update_multi": config.update_multi,
            "width_mode": config.width_mode,
            "enable_forward_interaction": config.enable_forward_interaction,
            "summary": {
                "content": config.summary.content,
                "i18n_content": config.summary.i18n_content
            }
        })
    except Exception as e:
        print(f"JSON serialization failed: {e}")
        return False

    req = SettingsCardRequest.builder() \
        .card_id(card_id) \
        .request_body(SettingsCardRequestBody.builder()
                      .settings(config_data)
                      .uuid("191857678434")
                      .sequence(1)
                      .build()) \
        .build()

    resp = client.cardkit.v1.card.settings(req)
    if not resp.success():
        print(f"Error updating card config: {resp.code}, {resp.msg}")
        return False

    lark.logger.info(lark.JSON.marshal(resp, indent=4))
    return True


def update_card_content(client: lark.Client, card_id: str, content: str) -> bool:
    req = ContentCardElementRequest.builder() \
        .card_id(card_id) \
        .element_id("elem_1") \
        .request_body(ContentCardElementRequestBody.builder()
                      .uuid("191857678434")
                      .content(content)
                      .sequence(1)
                      .build()) \
        .build()

    resp = client.cardkit.v1.card_element.content(req)
    if not resp.success():
        print(f"Error updating card content: {resp.code}, {resp.msg}")
        return False

    lark.logger.info(
        f"Card content updated success")
    return True
