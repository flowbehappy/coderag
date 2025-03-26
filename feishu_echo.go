package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
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
        "streaming_config": {
            "print_step": {
                "default": 1
            },
            "print_frequency_ms": {
                "default": 70
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
            {
                "tag": "markdown",
                "content": "AI generated content",
                "text_align": "left",
                "text_size": "normal_v2",
                "margin": "0px 0px 0px 0px",
                "element_id": "elem_1"
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
                                    "content": "Upvote"
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": "upvote"
                                    }
                                ],
                                "margin": "0px 0px 0px 0px"
                            },
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": "Downvote"
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": "downvote"
                                    }
                                ],
                                "margin": "0px 0px 0px 0px"
                            },
                            {
                                "tag": "button",
                                "text": {
                                    "tag": "plain_text",
                                    "content": "Regenerate"
                                },
                                "type": "default",
                                "width": "default",
                                "size": "medium",
                                "behaviors": [
                                    {
                                        "type": "callback",
                                        "value": "regen"
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

func updateCard(client *lark.Client, cardId string) error {

	updateContent := `飞书emoji :OK::THUMBSUP:
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
JSON
{"This is": "JSON demo"}

inline-code
# 一级标题
## 二级标题
> 这是一段引用

 | Syntax | Description |
| -------- | -------- |
| Header | Title |
| Paragraph | Text |"`
	req := larkcardkit.NewContentCardElementReqBuilder().
		CardId(cardId).
		ElementId(`elem_1`).
		Body(larkcardkit.NewContentCardElementReqBodyBuilder().
			Uuid(`191857678434`).
			Content(updateContent).
			Sequence(1).
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

// TODO: We cannot get the messages with interactive MsgType.
// https://open.feishu.cn/search?from=header&page=1&pageSize=10&q=%E8%8E%B7%E5%8F%96%E5%8D%A1%E7%89%87%E5%86%85%E5%AE%B9&topicFilter=
func getAllMessagesInThread(client *lark.Client, threadId string) error {
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("thread").
		ContainerId(threadId).
		SortType("ByCreateTimeAsc").
		// PageSize(20).
		// PageToken(`GxmvlNRvP0NdQZpa7yIqf_Lv_QuBwTQ8tXkX7w-irAghVD_TvuYd1aoJ1LQph86O-XImC4X9j9FhUPhXQDvtrQ==`).
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

func main() {
	app_id := os.Getenv("FEISHU_APP_ID")
	app_secret := os.Getenv("FEISHU_APP_SECRET")

	/**
	 * 创建 LarkClient 对象，用于请求OpenAPI。
	 * Create LarkClient object for requesting OpenAPI
	 */
	client := lark.NewClient(app_id, app_secret)

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
				cardId, err := createCard(client)
				if err != nil {
					fmt.Println("Create card failed, error:", err)
					return nil
				}
				err = sendCardToUser(client, cardId, *event.Event.Message.MessageId, threadId != nil)
				if err != nil {
					fmt.Println("Send card failed, error:", err)
					return nil
				}

				// Update card content interactively
				go func() {
					time.Sleep(1 * time.Second)
					updateCard(client, cardId)
				}()
			}

			return nil
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
