import os
from typing import Optional, Generator, Tuple
import json
import boto3
import re


DEFAULT_MODEL = "us.deepseek.r1-v1:0"

# TODO: support more models
MODEL_MAP = {
    "claude-3-sonnet": "anthropic.claude-3-sonnet-20240229-v1:0",
    "claude-3-5-sonnet": "anthropic.claude-3-5-sonnet-20240620-v1:0",
    "claude-3-5-sonnet-v2": "anthropic.claude-3-5-sonnet-20241022-v2:0",
    "claude-3-7-sonnet": "anthropic.claude-3-7-sonnet-20250219-v1:0",
    "deepseek-R1": "us.deepseek.r1-v1:0",
}


class BedrockProvider:
    @staticmethod
    def get_credentials() -> dict:
        required_vars = ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]
        missing_vars = [var for var in required_vars if not os.getenv(var)]
        if missing_vars:
            raise ValueError(
                f"Missing required AWS environment variables: {', '.join(missing_vars)}"
            )
        region = os.getenv("AWS_DEFAULT_REGION", "us-east-1")
        return {
            "aws_access_key_id": os.getenv("AWS_ACCESS_KEY_ID"),
            "aws_secret_access_key": os.getenv("AWS_SECRET_ACCESS_KEY"),
            "region_name": region,
        }

    @staticmethod
    def is_configured() -> bool:
        try:
            BedrockProvider.get_credentials()
            return True
        except ValueError:
            return False

    def __init__(self, model: str, **kwargs):
        credentials = self.get_credentials()
        self.client = boto3.client("bedrock-runtime", **credentials)

        self.model = MODEL_MAP.get(model, DEFAULT_MODEL)

    def generate(
        self, prompt: str, system_prompt: Optional[str] = None, **kwargs
    ) -> Optional[str]:
        messages = [{"role": "user", "content": [{"text": prompt}]}]
        if system_prompt:
            response = self.client.converse(
                modelId=self.model,
                inferenceConfig={
                    "maxTokens": 30000,
                    "temperature": 0.6,
                },
                system=[{"text": system_prompt}],
                messages=messages,
            )
        else:
            response = self.client.converse(
                modelId=self.model,
                inferenceConfig={
                    "maxTokens": 30000,
                    "temperature": 0.6,
                },
                messages=messages,
            )
        answer = None
        reasoning = None
        for message in response["output"]["message"]["content"]:
            if "text" in message:
                answer = message["text"]
            elif "reasoningContent" in message:
                reasoning = message["reasoningContent"]["reasoningText"]["text"]
        if reasoning:
            return f"<think>{reasoning}</think>\n{answer}"
        else:
            return answer

    def generate_stream(
        self, prompt: str, system_prompt: Optional[str] = None, **kwargs
    ) -> Generator[str, None, None]:
        if system_prompt:
            messages = [
                {
                    "role": "system",
                    "content": [
                        {"type": "text", "text": system_prompt + "\n" + prompt}
                    ],
                }
            ]
        else:
            messages = [{"role": "user", "content": [
                {"type": "text", "text": prompt}]}]
        try:
            request_body = {
                "anthropic_version": "bedrock-2023-05-31",
                "max_tokens": 8192,
                "messages": messages,
            }
            streaming_response = self._retry_with_exponential_backoff(
                self.client.invoke_model_with_response_stream,
                modelId=self.model,
                body=json.dumps(request_body),
            )
            for event in streaming_response["body"]:
                chunk = json.loads(event["chunk"]["bytes"])
                if chunk["type"] == "content_block_delta":
                    yield chunk["delta"].get("text", "")
        except Exception as e:
            print(f"Error during Bedrock streaming: {e}")
            yield f"Error: {str(e)}"


def split_think_content(text: str) -> Tuple[str, str]:
    think_pattern = re.compile(
        r'<think>\s*(.*?)\s*</think>',
        re.DOTALL | re.IGNORECASE
    )

    think_matches = think_pattern.findall(text)

    think_content = think_matches[-1].strip() if think_matches else ""

    reply_content = think_pattern.sub('', text).strip()

    if not think_matches:
        reply_content = text.strip()

    think_content = think_content if think_content else "[无思考内容]"
    reply_content = reply_content if reply_content else "[无回复内容]"

    return think_content, reply_content


async def request(data: str | None, model: str = DEFAULT_MODEL) -> str:
    if data is None:
        return "please input text!"
    p = BedrockProvider(model)
    result = p.generate(data)
    think_content, reply_content = split_think_content(result)
    return reply_content


model = "us.deepseek.r1-v1:0"
if __name__ == "__main__":
    prompt = "who are you"
    p = BedrockProvider(model)
    result = p.generate("prompt")
    print("reviced result from LLM", result)
