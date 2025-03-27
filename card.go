package main

import (
	"context"
	"encoding/json"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
)

// go embed card.json
var cardContent string

// see https://open.feishu.cn/document/uAjLw4CM/ukzMukzMukzM/feishu-cards/card-json-v2-structure
type cardConfig struct {
	StreamingMode            bool   `json:"streaming_mode,omitempty"`
	EnableForward            bool   `json:"enable_forward,omitempty"`
	UpdateMulti              bool   `json:"update_multi,omitempty"`
	WidthMode                string `json:"width_mode,omitempty"`
	EnableForwardInteraction bool   `json:"enable_forward_interaction,omitempty"`
	Summary                  Summary
	//
}

func (c *cardConfig) SetStreamingMode(mode bool) {
	c.StreamingMode = mode
}

func (c *cardConfig) UpdateSummary(summary Summary) {
	c.Summary = summary
}

type Summary struct {
	Content     string            `json:"content,omitempty"`
	I18nContent map[string]string `json:"i18n_content,omitempty"` // 摘要信息的多语言配置。了解支持的所有语种。参考配置卡片多语言文档。
}

func createCard(client *lark.Client) (string, error) {
	// 创建请求对象
	req := larkcardkit.NewCreateCardReqBuilder().
		Body(larkcardkit.NewCreateCardReqBodyBuilder().
			Type(`card_json`).
			Data(cardContent).
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

// 1. 关闭卡片的流式更新模式以交互更新的方式更新卡片,将 streaming_mode 字段值设置为 false 关闭流式
// 2. [生成中] 的摘要文本由卡片 JSON 中的 summary 属性控制
func updateCardConfig(client *lark.Client, cardId string, config cardConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	req := larkcardkit.NewSettingsCardReqBuilder().
		CardId(cardId).
		Body(larkcardkit.NewSettingsCardReqBodyBuilder().
			Settings(string(data)).
			Uuid(`191857678434`).
			Sequence(1).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Cardkit.V1.Card.Settings(context.Background(), req)

	// 处理错误
	if err != nil {
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

func updateCardContent(client *lark.Client, cardId string, result string) error {
	req := larkcardkit.NewContentCardElementReqBuilder().
		CardId(cardId).
		ElementId(`elem_1`).
		Body(larkcardkit.NewContentCardElementReqBodyBuilder().
			Uuid(`191857678434`).
			Content(result).
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
