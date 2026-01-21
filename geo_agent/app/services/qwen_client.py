"""
Qwen API client using DashScope SDK
"""
import time
import json
from typing import Optional, Dict, Any, AsyncGenerator
from dashscope import Generation
from dashscope.api_entities.dashscope_response import GenerationResponse
import dashscope

from app.core.config import config
from app.core.logger import get_logger
from app.models.dashscope import (
    DashScopeMessage,
    DashScopeInput,
    DashScopeParameters,
    DashScopeUsage
)

logger = get_logger(__name__)

class QwenClient:
    """
    Qwen API client wrapper for DashScope
    """
    
    def __init__(self):
        self.api_key = config.qwen.api_key
        self.model = config.qwen.model
        self.timeout = config.qwen.timeout
        self.max_retries = config.qwen.max_retries
        
        # Set DashScope API key
        dashscope.api_key = self.api_key
        
        if not self.api_key:
            logger.warning("DASHSCOPE_API_KEY not set")
    
    async def chat_completion(
        self,
        messages: list[Dict[str, str]],
        temperature: Optional[float] = None,
        top_p: Optional[float] = None,
        max_tokens: Optional[int] = None,
        stop: Optional[list[str]] = None,
        stream: bool = False,
        **kwargs
    ) -> Dict[str, Any]:
        """
        Call Qwen chat completion API
        
        Args:
            messages: List of chat messages
            temperature: Sampling temperature
            top_p: Nucleus sampling parameter
            max_tokens: Maximum tokens to generate
            stop: Stop sequences
            stream: Enable streaming
            
        Returns:
            Response dict with output and usage
        """
        start_time = time.time()
        
        # Build parameters
        parameters = {}
        if temperature is not None:
            parameters['temperature'] = temperature
        if top_p is not None:
            parameters['top_p'] = top_p
        if max_tokens is not None:
            parameters['max_tokens'] = max_tokens
        if stop:
            parameters['stop'] = stop
        
        # Set result format
        parameters['result_format'] = 'message'
        
        # Add incremental output for streaming
        if stream:
            parameters['incremental_output'] = True
        
        try:
            logger.info(
                "calling_qwen_api",
                model=self.model,
                message_count=len(messages),
                stream=stream
            )
            
            # Print detailed DashScope request info
            print(f"\n{'='*80}")
            print(f"🚀 调用DashScope API")
            print(f"{'='*80}")
            print(f"模型: {self.model}")
            print(f"流式: {stream}")
            print(f"消息数: {len(messages)}")
            print(f"参数: {json.dumps(parameters, ensure_ascii=False, indent=2)}")
            print(f"\nDashScope消息:")
            for i, msg in enumerate(messages, 1):
                print(f"  [{i}] {msg.get('role', 'unknown')}: {str(msg.get('content', ''))[:200]}{'...' if len(str(msg.get('content', ''))) > 200 else ''}")
            print(f"{'='*80}\n")
            
            if stream:
                # Streaming mode
                return await self._stream_chat(messages, parameters)
            else:
                # Non-streaming mode
                response = Generation.call(
                    model=self.model,
                    messages=messages,
                    **parameters
                )
                
                latency = (time.time() - start_time) * 1000
                
                if response.status_code != 200:
                    error_msg = f"DashScope API error: {response.code} - {response.message}"
                    logger.error(
                        "qwen_api_error",
                        status_code=response.status_code,
                        code=response.code,
                        message=response.message,
                        latency_ms=latency
                    )
                    raise Exception(error_msg)
                
                # Parse response
                result = self._parse_response(response, latency)
                
                logger.info(
                    "qwen_api_success",
                    request_id=response.request_id,
                    tokens=result['usage']['total_tokens'],
                    latency_ms=latency
                )
                
                # Print detailed DashScope response info
                print(f"\n{'='*80}")
                print(f"✅ DashScope API响应")
                print(f"{'='*80}")
                print(f"Request ID: {response.request_id}")
                print(f"响应内容: {result['content'][:500]}{'...' if len(result['content']) > 500 else ''}")
                print(f"完成原因: {result['finish_reason']}")
                print(f"\nToken使用:")
                print(f"  输入tokens: {result['usage']['input_tokens']}")
                print(f"  输出tokens: {result['usage']['output_tokens']}")
                print(f"  总tokens: {result['usage']['total_tokens']}")
                print(f"耗时: {latency:.2f}ms")
                print(f"{'='*80}\n")
                
                return result
                
        except Exception as e:
            latency = (time.time() - start_time) * 1000
            logger.error(
                "qwen_api_exception",
                error=str(e),
                latency_ms=latency
            )
            raise
    
    async def _stream_chat(
        self,
        messages: list[Dict[str, str]],
        parameters: Dict[str, Any]
    ) -> AsyncGenerator[Dict[str, Any], None]:
        """
        Handle streaming chat completion
        """
        try:
            responses = Generation.call(
                model=self.model,
                messages=messages,
                stream=True,
                **parameters
            )
            
            for response in responses:
                if response.status_code == 200:
                    yield self._parse_stream_chunk(response)
                else:
                    error_msg = f"Stream error: {response.code} - {response.message}"
                    logger.error("qwen_stream_error", error=error_msg)
                    raise Exception(error_msg)
                    
        except Exception as e:
            logger.error("qwen_stream_exception", error=str(e))
            raise
    
    def _parse_response(self, response: GenerationResponse, latency_ms: float) -> Dict[str, Any]:
        """
        Parse DashScope response to internal format
        """
        output = response.output
        usage = response.usage
        
        # Extract message content
        content = ""
        finish_reason = "stop"
        
        if hasattr(output, 'choices') and output.choices:
            choice = output.choices[0]
            if hasattr(choice, 'message'):
                content = choice.message.get('content', '')
            if hasattr(choice, 'finish_reason'):
                finish_reason = choice.finish_reason
        elif hasattr(output, 'text'):
            content = output.text
        
        # Build usage
        usage_dict = {
            "input_tokens": getattr(usage, 'input_tokens', 0),
            "output_tokens": getattr(usage, 'output_tokens', 0),
            "total_tokens": getattr(usage, 'total_tokens', 0)
        }
        
        return {
            "request_id": response.request_id,
            "content": content,
            "finish_reason": finish_reason,
            "usage": usage_dict,
            "latency_ms": latency_ms
        }
    
    def _parse_stream_chunk(self, response: GenerationResponse) -> Dict[str, Any]:
        """
        Parse streaming chunk
        """
        output = response.output
        
        content = ""
        finish_reason = None
        
        if hasattr(output, 'choices') and output.choices:
            choice = output.choices[0]
            if hasattr(choice, 'message'):
                content = choice.message.get('content', '')
            if hasattr(choice, 'finish_reason'):
                finish_reason = choice.finish_reason
        
        return {
            "request_id": response.request_id,
            "content": content,
            "finish_reason": finish_reason
        }
    
    def validate_api_key(self) -> bool:
        """
        Validate if API key is configured
        """
        return bool(self.api_key and self.api_key != "")

# Global client instance
qwen_client = QwenClient()
