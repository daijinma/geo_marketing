"""
Format converter between OpenAI and DashScope
"""
from typing import List, Dict, Any, Optional
import time

from app.models.openai import (
    ChatCompletionRequest,
    ChatCompletionResponse,
    ChatMessage,
    ChatChoice,
    Usage,
    ChatCompletionChunk,
    ChatCompletionChunkChoice,
    ChatCompletionChunkDelta
)
from app.utils.helpers import (
    generate_chat_completion_id,
    get_current_timestamp
)

class FormatConverter:
    """
    Convert between OpenAI and DashScope formats
    """
    
    @staticmethod
    def openai_to_dashscope_messages(
        messages: List[ChatMessage]
    ) -> List[Dict[str, str]]:
        """
        Convert OpenAI messages to DashScope format
        
        OpenAI: [{"role": "user", "content": "..."}]
        DashScope: [{"role": "user", "content": "..."}]
        """
        dashscope_messages = []
        
        for msg in messages:
            # DashScope uses same format but may have different constraints
            dashscope_msg = {
                "role": msg.role,
                "content": msg.content
            }
            dashscope_messages.append(dashscope_msg)
        
        return dashscope_messages
    
    @staticmethod
    def dashscope_to_openai_response(
        qwen_response: Dict[str, Any],
        original_request: ChatCompletionRequest,
        request_id: str
    ) -> ChatCompletionResponse:
        """
        Convert DashScope response to OpenAI format
        
        Args:
            qwen_response: Response from Qwen API
            original_request: Original OpenAI request
            request_id: Request ID
            
        Returns:
            OpenAI ChatCompletionResponse
        """
        content = qwen_response.get("content", "")
        finish_reason = qwen_response.get("finish_reason", "stop")
        usage_data = qwen_response.get("usage", {})
        
        # Build OpenAI response
        response = ChatCompletionResponse(
            id=generate_chat_completion_id(),
            object="chat.completion",
            created=get_current_timestamp(),
            model=original_request.model,
            choices=[
                ChatChoice(
                    index=0,
                    message=ChatMessage(
                        role="assistant",
                        content=content
                    ),
                    finish_reason=finish_reason
                )
            ],
            usage=Usage(
                prompt_tokens=usage_data.get("input_tokens", 0),
                completion_tokens=usage_data.get("output_tokens", 0),
                total_tokens=usage_data.get("total_tokens", 0)
            )
        )
        
        return response
    
    @staticmethod
    def dashscope_to_openai_stream_chunk(
        qwen_chunk: Dict[str, Any],
        model: str,
        chunk_id: str
    ) -> ChatCompletionChunk:
        """
        Convert DashScope streaming chunk to OpenAI format
        
        Args:
            qwen_chunk: Streaming chunk from Qwen
            model: Model name
            chunk_id: Chunk ID
            
        Returns:
            OpenAI ChatCompletionChunk
        """
        content = qwen_chunk.get("content", "")
        finish_reason = qwen_chunk.get("finish_reason")
        
        # First chunk should include role
        delta = ChatCompletionChunkDelta(content=content)
        if content and not finish_reason:
            # Regular content chunk
            delta.content = content
        elif finish_reason:
            # Last chunk
            delta.content = ""
        
        chunk = ChatCompletionChunk(
            id=chunk_id,
            object="chat.completion.chunk",
            created=get_current_timestamp(),
            model=model,
            choices=[
                ChatCompletionChunkChoice(
                    index=0,
                    delta=delta,
                    finish_reason=finish_reason
                )
            ]
        )
        
        return chunk
    
    @staticmethod
    def extract_openai_parameters(request: ChatCompletionRequest) -> Dict[str, Any]:
        """
        Extract parameters from OpenAI request for DashScope
        
        Maps OpenAI parameters to DashScope equivalents
        """
        params = {}
        
        # Direct mappings
        if request.temperature is not None:
            params['temperature'] = request.temperature
        
        if request.top_p is not None:
            params['top_p'] = request.top_p
        
        if request.max_tokens is not None:
            params['max_tokens'] = request.max_tokens
        
        # Stop sequences
        if request.stop:
            if isinstance(request.stop, str):
                params['stop'] = [request.stop]
            else:
                params['stop'] = request.stop
        
        # Stream mode
        if request.stream:
            params['stream'] = True
        
        return params

# Global converter instance
converter = FormatConverter()
