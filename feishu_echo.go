package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

type Text struct {
	Text string `json:"text,omitempty"`
}

type MessagePostContent struct {
	Title   string              `json:"title,omitempty"`
	Content MessagePostElements `json:"content,omitempty"`
	FromApp bool
}

type MessagePostElement interface{}

type MessagePostText struct {
	Tag   string   `json:"tag"`
	Text  string   `json:"text"`
	Style []string `json:"style"`
}

type MessagePostLink struct {
	Tag   string   `json:"tag"`
	Href  string   `json:"href"`
	Text  string   `json:"text"`
	Style []string `json:"style"`
}

type MessagePostAt struct {
	Tag      string   `json:"tag"`
	UserId   string   `json:"user_id"`
	UserName string   `json:"user_name"`
	Style    []string `json:"style"`
}

type MessagePostImage struct {
	Tag      string `json:"tag"`
	ImageKey string `json:"image_key"`
}

type MessagePostMedia struct {
	Tag      string `json:"tag"`
	FileKey  string `json:"file_key"`
	ImageKey string `json:"image_key"`
}

type MessagePostEmotion struct {
	Tag       string `json:"tag"`
	EmojiType string `json:"emoji_type"`
}

type MessagePostHr struct {
	Tag string `json:"tag"`
}

type MessagePostCode struct {
	Tag      string `json:"tag"`
	Language string `json:"language"`
	Text     string `json:"text"`
}

type MessagePostElements [][]MessagePostElement

func (e *MessagePostElements) UnmarshalJSON(data []byte) error {
	var rawRows [][]json.RawMessage
	if err := json.Unmarshal(data, &rawRows); err != nil {
		return err
	}

	*e = make([][]MessagePostElement, len(rawRows))
	for i, rawRow := range rawRows {
		row := make([]MessagePostElement, len(rawRow))
		for j, rawElem := range rawRow {
			var wrapper struct {
				Tag string `json:"tag"`
			}
			if err := json.Unmarshal(rawElem, &wrapper); err != nil {
				return err
			}

			switch wrapper.Tag {
			case "text":
				var textElem MessagePostText
				if err := json.Unmarshal(rawElem, &textElem); err != nil {
					return err
				}
				row[j] = &textElem
			case "a":
				var linkElem MessagePostLink
				if err := json.Unmarshal(rawElem, &linkElem); err != nil {
					return err
				}
				row[j] = &linkElem
			case "at":
				var atElem MessagePostAt
				if err := json.Unmarshal(rawElem, &atElem); err != nil {
					return err
				}
				row[j] = &atElem
			case "img":
				var imgElem MessagePostImage
				if err := json.Unmarshal(rawElem, &imgElem); err != nil {
					return err
				}
				row[j] = &imgElem
			case "media":
				var mediaElem MessagePostMedia
				if err := json.Unmarshal(rawElem, &mediaElem); err != nil {
					return err
				}
				row[j] = &mediaElem
			case "emotion":
				var emotionElem MessagePostEmotion
				if err := json.Unmarshal(rawElem, &emotionElem); err != nil {
					return err
				}
				row[j] = &emotionElem
			case "hr":
				var hrElem MessagePostHr
				if err := json.Unmarshal(rawElem, &hrElem); err != nil {
					return err
				}
				row[j] = &hrElem
			case "code_block":
				var codeElem MessagePostCode
				if err := json.Unmarshal(rawElem, &codeElem); err != nil {
					return err
				}
				row[j] = &codeElem
			default:
				return fmt.Errorf("Unknown tag: %s", wrapper.Tag)
			}
		}
		(*e)[i] = row
	}
	return nil
}

func getTextAndCode(contents []MessagePostContent) string {
	var result strings.Builder

	for i, content := range contents {
		// Add header based on FromApp flag
		if content.FromApp {
			result.WriteString("From AI:\n")
		} else {
			result.WriteString("From User:\n")
		}

		// Process each row of elements
		for j, row := range content.Content {
			var rowText string

			// Extract text from each element in the row
			for _, elem := range row {
				switch e := elem.(type) {
				case *MessagePostText:
					rowText += e.Text
				case *MessagePostCode:
					rowText += e.Text
				}
			}

			// Add the row text if not empty
			if rowText != "" {
				result.WriteString(rowText)

				// Add newline between rows (but not after the last row)
				if j < len(content.Content)-1 {
					result.WriteString("\n")
				}
			}
		}

		// Add empty line between contents (but not after the last content)
		if i < len(contents)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

var card_content = `
{
    "schema": "2.0",
    "config": {
        "update_multi": true,
        "streaming_mode": true,
		"summary": {"content": "Generating..."},
        "streaming_config": {
            "print_step": {
                "default": 1
            },
            "print_frequency_ms": {
                "default": 10
            },
            "print_strategy": "delay"
        },
        "style": {
            "text_size": {
                "normal_v2": {
                    "default": "normal",
                    "pc": "normal",
                    "mobile": "heading"
                }
            }
        }
    },
    "body": {
        "direction": "vertical",
        "padding": "12px 12px 12px 12px",
        "elements": [
            {"tag": "collapsible_panel",  "expanded": false, 
				"header": { "title": {"tag": "plain_text", "content": "Thinking..."}, "icon": {"tag": "standard_icon", "token": "down-small-ccm_outlined","size": "16px 16px"},"icon_position": "right"},
				"elements":[{
                "tag": "markdown",
                "content": "",
                "text_align": "left",
                "text_size": "normal_v2",
                "margin": "0px 0px 0px 0px",
                "element_id": "thinking"
            }]},
            {
                "tag": "markdown",
                "content": "",
                "text_align": "left",
                "text_size": "normal_v2",
                "margin": "0px 0px 0px 0px",
                "element_id": "result"
			},
            {
                "tag": "column_set",
                "horizontal_align": "left",
                "columns": [
                    {
                        "tag": "column",
                        "width": "auto",
                        "elements": [
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": ""
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "icon": {
                                    "tag": "standard_icon",
                                    "token": "like_filled"
                                },
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": {
                                            "action": "upvote"
                                        }
                                    }
                                ],
                                "margin": "0px 0px 0px 0px"
                            },
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": ""
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "icon": {
                                    "tag": "standard_icon",
                                    "token": "thumbdown_outlined"
                                },
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": {
                                            "action": "downvote"
                                        }
                                    }
                                ],
                                "margin": "0px 0px 0px 0px"
                            },
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": ""
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "icon": {
                                    "tag": "standard_icon",
                                    "token": "refresh_outlined"
                                },
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": {
                                            "action": "regen"
                                        }
                                    }
                                ],
                                "margin": "0px 0px 0px 0px"
                            }
                        ],
                        "direction": "horizontal",
                        "vertical_spacing": "8px",
                        "horizontal_align": "left",
                        "vertical_align": "top"
                    }
                ],
                "element_id": "elem_2"
            }
        ]
    }
}
`

func createCard(client *lark.Client) (string, error) {
	// 创建请求对象
	req := larkcardkit.NewCreateCardReqBuilder().
		Body(larkcardkit.NewCreateCardReqBodyBuilder().
			Type(`card_json`).
			Data(card_content).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Cardkit.V1.Card.Create(context.Background(), req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return "", err
	}

	respJson, _ := json.Marshal(resp)
	fmt.Println("Card reply:", string(respJson))
	return *resp.Data.CardId, nil
}

func sendCardToUser(client *lark.Client, cardId string, messageId string, isThread bool) error {
	sendContent := `{"type":"card","data":{"card_id":"` + cardId + `"}}`
	// Using message API to reply the message
	resp, err := client.Im.Message.Reply(context.Background(), larkim.NewReplyMessageReqBuilder().
		MessageId(messageId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			ReplyInThread(isThread).
			Content(sendContent).
			Build()).
		Build())

	if err != nil || !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return err
	}

	respJson, _ := json.Marshal(resp)
	fmt.Println("Reply message response:", string(respJson))
	return nil
}

var thiningContent = `
嗯，用户问的是MCP，也就是Model Context Protocol是什么。首先，我需要确认自己对这个概念的理解是否正确。可能之前看过相关资料，但需要回忆一下。

首先，Model Context Protocol听起来像是一个协议或者框架，可能与机器学习模型的上下文管理有关。根据之前的知识，MCP可能涉及模型训练、部署中的上下文信息管理，比如参数、环境配置、数据流等。需要确认具体定义和应用场景。

....
`

var result = `
飞书emoji :OK::THUMBSUP:
*斜体* **粗体** ~~删除线~~ 
<font color='red'>这是红色文本</font>
<text_tag color='blue'>标签</text_tag>
<number_tag>1</number_tag>
[文字链接](https://open.feishu.cn/server-docs/im-v1/message-reaction/emojis-introduce)
<link icon='chat_outlined' url='https://open.feishu.cn' pc_url='' ios_url='' android_url=''>带图标的链接</link>
<at id=all></at>
- 无序列表1
	- 无序列表 1.1
- 无序列表2
1. 有序列表1
	1. 有序列表 1.1
2. 有序列表2
` + "\n```JSON\n{" + `"This is": "JSON demo"}` + "\n```\n" + "`inline-code`\n" + `

# 一级标题
## 二级标题
> 这是一段引用

| Syntax | Description |
| -------- | -------- |
| Header | Title |
| Paragraph | Text |"`

var newResult = `New result`

func updateCardContent(client *lark.Client, cardId string, elementId string, content string, seqNum int) error {
	req := larkcardkit.NewContentCardElementReqBuilder().
		CardId(cardId).
		ElementId(elementId).
		Body(larkcardkit.NewContentCardElementReqBodyBuilder().
			Uuid(uuid.New().String()).
			Content(content).
			Sequence(seqNum).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Cardkit.V1.CardElement.Content(context.Background(), req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return err
	}

	// 业务处理
	fmt.Println("card update resp: ", larkcore.Prettify(resp))
	return nil
}

func updateCardSummary(client *lark.Client, cardId string, seqNum int) error {
	req := larkcardkit.NewSettingsCardReqBuilder().
		CardId(cardId).
		Body(larkcardkit.NewSettingsCardReqBodyBuilder().
			Settings(`{"config":{"summary":{"content":"Done"}}}`).
			Uuid(uuid.New().String()).
			Sequence(seqNum).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Cardkit.V1.Card.Settings(context.Background(), req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return err
	}

	// 业务处理
	fmt.Println(larkcore.Prettify(resp))
	return nil
}

// TODO: We cannot get the messages with interactive MsgType.
// https://open.feishu.cn/search?from=header&page=1&pageSize=10&q=%E8%8E%B7%E5%8F%96%E5%8D%A1%E7%89%87%E5%86%85%E5%AE%B9&topicFilter=
func getAllMessagesInThread(client *lark.Client, threadId string) error {
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("thread").
		ContainerId(threadId).
		SortType("ByCreateTimeAsc").
		Build()

	// Get all history messages in the thread
	resp, err := client.Im.V1.Message.List(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return nil
	}
	fmt.Println("======", larkcore.Prettify(resp))

	hisContents := []MessagePostContent{}
	for _, msg := range resp.Data.Items {
		// We only handle text and post messages
		content := MessagePostContent{}
		if *msg.MsgType == larkim.MsgTypeText {
			// For convience, we transform text into post
			text := Text{}
			err := json.Unmarshal([]byte(*msg.Body.Content), &text)
			if err != nil {
				fmt.Println("Unmarshal history messages failed, error:", err)
				return nil
			}
			content.Content = MessagePostElements{{&MessagePostText{Tag: "text", Text: text.Text, Style: nil}}}
		} else if *msg.MsgType == larkim.MsgTypePost {
			mc := *msg.Body.Content
			err := json.Unmarshal([]byte(mc), &content)
			if err != nil {
				fmt.Println("Unmarshal history messages failed, error:", err)
				return nil
			}
		} else {
			fmt.Println("Unsupported message type:", *msg.MsgType)
			continue
		}
		content.FromApp = *msg.Sender.SenderType == "app"
		hisContents = append(hisContents, content)
	}
	hms, _ := json.Marshal(hisContents)
	fmt.Println("History messages:", string(hms))
	fmt.Println("Text and code:", getTextAndCode(hisContents))
	return nil
}

func getMessage(client *lark.Client, messageId string) (*larkim.GetMessageResp, error) {
	req := larkim.NewGetMessageReqBuilder().
		MessageId(messageId).
		UserIdType(`open_id`).
		Build()

	// 发起请求
	resp, err := client.Im.V1.Message.Get(context.Background(), req)
	// 处理错误
	if err != nil || !resp.Success() {
		fmt.Println("Get message failed, error:", err)
		return nil, err
	}
	fmt.Println("Get message resp:", larkcore.Prettify(resp))
	return resp, nil
}

func main() {
	app_id := os.Getenv("FEISHU_APP_ID")
	app_secret := os.Getenv("FEISHU_APP_SECRET")

	/**
	 * 创建 LarkClient 对象，用于请求OpenAPI。
	 * Create LarkClient object for requesting OpenAPI
	 */
	client := lark.NewClient(app_id, app_secret)

	sendResult := func(client *lark.Client, messageId string, toThread bool) (string, error) {
		cardId, err := createCard(client)
		if err != nil {
			fmt.Println("Create card failed, error:", err)
			return "", err
		}
		err = sendCardToUser(client, cardId, messageId, toThread)
		if err != nil {
			fmt.Println("Send card failed, error:", err)
			return "", err
		}
		return cardId, nil
	}

	/**
	 * 注册事件处理器。
	 * Register event handler.
	 */
	eventHandler := dispatcher.NewEventDispatcher("", "").
		/**
		 * 注册接收消息事件，处理接收到的消息。
		 * Register event handler to handle received messages.
		 * https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/events/receive
		 */
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			fmt.Printf("[OnP2MessageReceiveV1 access], data: %s\n", larkcore.Prettify(event))
			switch *event.Event.Message.ChatType {
			case "group":
				ms := event.Event.Message.Mentions
				atMe := false
				for _, mention := range ms {
					if *mention.Name == "deephack" {
						atMe = true
						break
					}
				}
				if !atMe {
					return nil
				}
			case "p2p":
			case "topic_group":
				return nil
			}

			threadId := event.Event.Message.ThreadId
			if threadId != nil {
				getAllMessagesInThread(client, *event.Event.Message.ThreadId)
			}

			{
				cardId, err := sendResult(client, *event.Event.Message.MessageId, threadId != nil)
				if err != nil {
					return nil
				}
				// Update card content interactively
				updateCardContent(client, cardId, "thinking", thiningContent, 1)
				updateCardContent(client, cardId, "result", result, 2)
				updateCardSummary(client, cardId, 3)
			}

			return nil
		}).
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			fmt.Printf("[ OnP2CardActionTrigger access ], data: %s\n", larkcore.Prettify(event))
			action := event.Event.Action.Value["action"]
			if action == "upvote" {
				return &callback.CardActionTriggerResponse{
					Toast: &callback.Toast{
						Type:    "success",
						Content: "Upvoted",
					},
				}, nil
			} else if action == "downvote" {
				return &callback.CardActionTriggerResponse{
					Toast: &callback.Toast{
						Type:    "success",
						Content: "Downvoted",
					},
				}, nil
			} else if action == "regen" {
				// Use another goroutine to avoid blocking the main thread
				go func() {
					messageId := event.Event.Context.OpenMessageID
					message, err := getMessage(client, messageId)
					if err != nil {
						fmt.Printf("getMessage failed!", err)
						return
					}
					threadId := message.Data.Items[0].ThreadId

					cardId, err := sendResult(client, messageId, threadId != nil)

					updateCardContent(client, cardId, "thinking", thiningContent, 1)
					updateCardContent(client, cardId, "result", newResult, 2)
					updateCardSummary(client, cardId, 3)
				}()
			}

			return &callback.CardActionTriggerResponse{
				Toast: &callback.Toast{
					Type:    "success",
					Content: "Regenerating",
				},
			}, nil
		})

	/**
	 * 启动长连接，并注册事件处理器。
	 * Start long connection and register event handler.
	 */
	cli := larkws.NewClient(app_id, app_secret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)
	err := cli.Start(context.Background())
	if err != nil {
		panic(err)
	}
}
