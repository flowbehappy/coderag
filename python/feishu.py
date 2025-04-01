import os
import json
import asyncio

from dotenv import load_dotenv
import lark_oapi as lark
from lark_oapi.api.application.v6 import *
from lark_oapi.api.im.v1 import *
from lark_oapi.event.callback.model.p2_card_action_trigger import (
    P2CardActionTrigger,
    P2CardActionTriggerResponse,
)

import card
from type import *
from model import request
from database import DB


def parse_message_post_content(data: str) -> MessagePostContent:
    raw = json.loads(data)
    content = MessagePostContent()
    content.title = raw.get("title", "")

    for row in raw.get("content", []):
        elements = []
        for elem in row:
            tag = elem.get("tag")
            if tag == "text":
                elements.append(MessagePostText(**elem))
            elif tag == "a":
                elements.append(MessagePostLink(**elem))
            elif tag == "at":
                elements.append(MessagePostAt(**elem))
            elif tag == "img":
                elements.append(MessagePostImage(**elem))
            elif tag == "media":
                elements.append(MessagePostMedia(**elem))
            elif tag == "emotion":
                elements.append(MessagePostEmotion(**elem))
            elif tag == "hr":
                elements.append(MessagePostHr(**elem))
            elif tag == "code_block":
                elements.append(MessagePostCode(**elem))
        content.content.append(elements)
    return content


def send_card_to_user(client: lark.Client, card_id: str, message_id: str) -> None:
    '''
    send card reply in thread
    '''
    send_content = json.dumps({
        "type": "card",
        "data": {"card_id": card_id}
    })

    resp = client.im.v1.message.reply(ReplyMessageRequest.builder()
                                      .message_id(message_id)
                                      .request_body(ReplyMessageRequestBody.builder()
                                                    .msg_type("interactive")
                                                    .reply_in_thread(True)
                                                    .content(send_content)
                                                    .build())
                                      .build())

    if not resp.success():
        lark.logger.error(f"Error sending card: {resp.code}, {resp.msg}")
    else:
        lark.logger.info(
            f"Card sent successfully")
    return resp.data.message_id


def get_message_by_message_id(message_id: str):
    req = GetMessageRequest.builder() \
        .message_id(message_id) \
        .build()
    resp = client.request(req)
    resp = client.im.v1.message.get(req)
    if not resp.success():
        lark.logger.error(f"Error getting messages: {resp.code}, {resp.msg}")
        return None
    lark.logger.info("recived message by message id: %s nums:%d",
                     message_id, len(resp.data.items))
    if len(resp.data.items) < 1:
        return None
    return resp.data.items[0].body.content


def get_thread_id_by_message_id(message_id: str):
    req = GetMessageRequest.builder() \
        .message_id(message_id) \
        .build()
    resp = client.im.v1.message.get(req)
    if not resp.success():
        lark.logger.error(f"Error getting messages: {resp.code}, {resp.msg}")
        return None
    lark.logger.info("recived message by message id: %s nums:%d",
                     message_id, len(resp.data.items))
    if len(resp.data.items) < 1:
        return None
    item = resp.data.items[0]
    return item.thread_id, item.create_time


def get_text_and_code(contents: List[MessagePostContent]) -> List[Dict]:
    result = []
    for i, content in enumerate(contents):
        msg = dict()
        msg["role"] = "assistant" if content.from_app else "user"
        for j, row in enumerate(content.content):
            row_text = ""
            for elem in row:
                # add text and code
                if isinstance(elem, (MessagePostText, MessagePostCode)):
                    row_text += elem.text

        msg["content"] = [{"text": row_text}]
        result.append(msg)

    return result


def get_all_messages_in_thread(client: lark.Client, thread_id: str, create_time: str) -> List[Dict]:
    req = ListMessageRequest.builder() \
        .container_id_type("thread") \
        .container_id(thread_id) \
        .sort_type("ByCreateTimeAsc") \
        .end_time(create_time) \
        .build()
    resp = client.im.v1.message.list(req)
    if not resp.success():
        lark.logger.error(f"Error getting messages: {resp.code}, {resp.msg}")
        return

    his_contents = []
    for msg in resp.data.items:
        content = MessagePostContent()
        if msg.msg_type == "text":
            text = json.loads(msg.body.content)
            elem = MessagePostText(text=text.get("text", ""))
            content.content = [[elem]]
        elif msg.msg_type == "post":
            content = parse_message_post_content(msg.body.content)
        elif msg.msg_type == "interactive":
            continue

        content.from_app = msg.sender.sender_type == "app"
        his_contents.append(content)

    result = get_text_and_code(his_contents)
    lark.logger.info(f"get past messages: {lark.json.dumps(result)}")
    return result


def do_p2_im_message_receive_v1(data: P2ImMessageReceiveV1) -> None:
    chat_type = data.event.message.chat_type
    mentions = data.event.message.mentions
    if chat_type == "group":
        if not any(mention.name == "deephack" for mention in mentions):
            return

    res_content = ""
    if data.event.message.message_type == "text":
        res_content = json.loads(data.event.message.content)["text"]
    else:
        lark.logger.error("parse message failed, please send text message")
        return

    thread_id = data.event.message.thread_id
    message_id = data.event.message.message_id
    create_time = data.event.message.create_time

    # 获取历史消息
    past_result = [None]
    is_thread = thread_id is not None
    if is_thread:
        past_result = get_all_messages_in_thread(
            client, thread_id, create_time)

    card_id = card.create_card(client)
    print("create card", card_id)
    card_message_id = send_card_to_user(
        client, card_id, message_id)

    # 更新卡片
    # message_id msg_type content
    async def update_task():
        think_content, reply_content = await request(res_content, past_result[:-1])
        lark.logger.info(
            "recieve message from LLM\nthink_content: %s\nreply_content: %s", think_content, reply_content)
        card.update_card_content(
            client, card_id, "thinking", reply_content, seq=1)
        card.update_card_content(
            client, card_id, "result", reply_content, seq=2)
        card.update_card_config(
            client, card_id, CardConfig(summary=Summary(content="Done")), seq=3)

    asyncio.create_task(update_task())


# see https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/event-subscription-guide/callback-subscription/configure-callback-request-address
def do_p2_card_action_trigger(data: P2CardActionTrigger) -> P2CardActionTriggerResponse:
    action = data.event.action
    # Use action to distinguish different buttons. You can configure the action of the button in the card building tool.
    # Here, handle the situation where the user clicks the "Initiate Alarm" button on the welcome card.
    match action.value["action"]:
        case "refresh":
            message_id = data.event.context.open_message_id
            thread_id, create_time = get_thread_id_by_message_id(message_id)
            past_result = get_all_messages_in_thread(
                client, thread_id, create_time)
            card_id = card.create_card(client)
            send_card_to_user(
                client, card_id, message_id)

            async def update_task():
                last = past_result[-1]["content"][0]["text"]
                think_content, reply_content = await request(last)
                lark.logger.info(
                    "recieve message from LLM\nthink_content: %s\nreply_content: %s", think_content, reply_content)
                card.update_card_content(
                    client, card_id, "thinking", reply_content, seq=1)
                card.update_card_content(
                    client, card_id, "result", reply_content, seq=2)
                card.update_card_config(
                    client, card_id, CardConfig(summary=Summary(content="Done")), seq=3)
            asyncio.create_task(update_task())
    return P2CardActionTriggerResponse({})


if __name__ == "__main__":
    load_dotenv(verbose=True)
    app_id = os.getenv("APP_ID")
    app_secret = os.getenv("APP_SECRET")
    db = DB()
    client = lark.Client.builder().app_id(app_id).app_secret(app_secret).build()
    event_handler = lark.EventDispatcherHandler.builder("", "") \
        .register_p2_im_message_receive_v1(do_p2_im_message_receive_v1) \
        .register_p2_card_action_trigger(do_p2_card_action_trigger) \
        .build()

    ws_client = lark.ws.Client(
        app_id, app_secret, lark.LogLevel.INFO, event_handler)
    ws_client.start()
