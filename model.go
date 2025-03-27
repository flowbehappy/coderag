package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

const defaultRegion = "us-east-1"

const (
	deepSeekModelID  = "us.deepseek.r1-v1:0"            //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
	claudeV2ModelID  = "anthropic.claude-v2"            //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
	llama3_8BModelID = "meta.llama3-1-8b-instruct-v1:0" //https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html
)

const prompt = `%s`

func main() {
	result := request("who are you?")
	fmt.Println("response from LLM\n", result)
}

func request(data string) string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = defaultRegion
	}
	key := os.Getenv("AWS_ACCESS_KEY")
	secret := os.Getenv("AWS_SECRET")
	session := os.Getenv("AWS_SESSION")
	credentialProvider := credentials.NewStaticCredentialsProvider(key, secret, session)
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentialProvider),
	)
	if err != nil {
		log.Fatal(err)
	}

	brc := bedrockruntime.NewFromConfig(cfg)

	payload := Request{
		Prompt:            fmt.Sprintf(prompt, data),
		MaxTokensToSample: 200,
		// Temperature:       0.5,
		// TopK:              250,
		// TopP:              1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	output, err := brc.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		Body:        payloadBytes,
		ModelId:     aws.String(deepSeekModelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	})

	if err != nil {
		log.Fatal("failed to invoke model: ", err)
	}

	var resp Response

	err = json.Unmarshal(output.Body, &resp)

	if err != nil {
		log.Fatal("failed to unmarshal", err)
	}

	return resp.Completion
}

//request/response model

type Request struct {
	Prompt            string   `json:"prompt"`
	MaxTokensToSample int      `json:"max_tokens_to_sample"`
	Temperature       float64  `json:"temperature,omitempty"`
	TopP              float64  `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
}

type Response struct {
	Completion string `json:"completion"`
}
