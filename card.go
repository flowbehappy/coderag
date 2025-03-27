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

type cardConfig struct {
	StreamingMode            bool   `json:"streaming_mode"`
	EnableForward            bool   `json:"enable_forward"`
	UpdateMulti              bool   `json:"update_multi"`
	WidthMode                string `json:"width_mode"`
	EnableForwardInteraction bool   `json:"enable_forward_interaction"`
	//
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

func updateCardConfig(client *lark.Client, cardId string, config cardConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	req := larkcardkit.NewSettingsCardReqBuilder().
		CardId(cardId).
		Body(larkcardkit.NewSettingsCardReqBodyBuilder().
			Settings(data).
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
