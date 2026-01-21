"""
OpenAI /v1/chat/completions endpoint
"""
import time
from typing import Any
from fastapi import APIRouter, Request, HTTPException
from fastapi.responses import StreamingResponse
import json

from app.models.openai import (
    ChatCompletionRequest,
    ChatCompletionResponse,
    ErrorResponse,
    ErrorDetail
)
from app.services.qwen_client import qwen_client
from app.services.converter import converter
from app.core.logger import logger_service, get_logger
from app.utils.helpers import generate_request_id, generate_chat_completion_id

router = APIRouter()
logger = get_logger(__name__)

@router.post("/chat/completions")
async def create_chat_completion(
    request_data: ChatCompletionRequest,
    request: Request
):
    """
    OpenAI compatible chat completion endpoint
    
    Converts OpenAI format to DashScope format, calls Qwen API,
    and converts response back to OpenAI format.
    """
    request_id = generate_request_id()
    start_time = time.time()
    client_ip = request.client.host if request.client else "unknown"
    
    # Validate API key
    if not qwen_client.validate_api_key():
        logger.error("api_key_not_configured", request_id=request_id)
        raise HTTPException(
            status_code=500,
            detail="DASHSCOPE_API_KEY not configured"
        )
    
    try:
        # Convert OpenAI messages to DashScope format
        dashscope_messages = converter.openai_to_dashscope_messages(
            request_data.messages
        )
        
        # Extract parameters
        params = converter.extract_openai_parameters(request_data)
        
        # Log request
        request_dict = {
            "model": request_data.model,
            "messages": [msg.model_dump() for msg in request_data.messages],
            "temperature": request_data.temperature,
            "max_tokens": request_data.max_tokens,
            "stream": request_data.stream
        }
        
        logger.info(
            "chat_completion_request",
            request_id=request_id,
            model=request_data.model,
            message_count=len(request_data.messages),
            stream=request_data.stream
        )
        
        # Print detailed incoming request info
        print(f"\n{'='*80}")
        print(f"📨 收到OpenAI格式请求 - Request ID: {request_id}")
        print(f"{'='*80}")
        print(f"客户端IP: {client_ip}")
        print(f"模型: {request_data.model}")
        print(f"消息数: {len(request_data.messages)}")
        print(f"流式: {request_data.stream}")
        if request_data.temperature:
            print(f"温度: {request_data.temperature}")
        if request_data.max_tokens:
            print(f"最大tokens: {request_data.max_tokens}")
        print(f"\nOpenAI消息:")
        for i, msg in enumerate(request_data.messages, 1):
            print(f"  [{i}] {msg.role}: {msg.content[:200] if msg.content else 'N/A'}{'...' if msg.content and len(msg.content) > 200 else ''}")
        print(f"{'='*80}\n")
        
        # Handle streaming vs non-streaming
        if request_data.stream:
            return await _handle_streaming(
                request_id=request_id,
                request_data=request_data,
                dashscope_messages=dashscope_messages,
                params=params,
                client_ip=client_ip,
                request_dict=request_dict,
                start_time=start_time
            )
        else:
            return await _handle_non_streaming(
                request_id=request_id,
                request_data=request_data,
                dashscope_messages=dashscope_messages,
                params=params,
                client_ip=client_ip,
                request_dict=request_dict,
                start_time=start_time
            )
            
    except HTTPException:
        raise
    except Exception as e:
        latency_ms = (time.time() - start_time) * 1000
        error_msg = str(e)
        
        logger.error(
            "chat_completion_error",
            request_id=request_id,
            error=error_msg,
            latency_ms=latency_ms
        )
        
        # Log to qwen_calls.log
        logger_service.log_qwen_call(
            request_id=request_id,
            request=request_dict if 'request_dict' in locals() else {},
            error=error_msg,
            latency_ms=latency_ms,
            client_ip=client_ip
        )
        
        # Log error
        logger_service.log_error(
            error_type=type(e).__name__,
            error_message=error_msg,
            request_id=request_id,
            traceback=str(e)
        )
        
        raise HTTPException(
            status_code=500,
            detail=f"Internal server error: {error_msg}"
        )

async def _handle_non_streaming(
    request_id: str,
    request_data: ChatCompletionRequest,
    dashscope_messages: list,
    params: dict,
    client_ip: str,
    request_dict: dict,
    start_time: float
) -> ChatCompletionResponse:
    """Handle non-streaming chat completion"""
    
    # Call Qwen API
    qwen_response = await qwen_client.chat_completion(
        messages=dashscope_messages,
        **params
    )
    
    # Convert to OpenAI format
    openai_response = converter.dashscope_to_openai_response(
        qwen_response=qwen_response,
        original_request=request_data,
        request_id=request_id
    )
    
    # Calculate latency
    latency_ms = (time.time() - start_time) * 1000
    
    # Log to qwen_calls.log
    logger_service.log_qwen_call(
        request_id=request_id,
        request=request_dict,
        response={
            "content": openai_response.choices[0].message.content,
            "finish_reason": openai_response.choices[0].finish_reason
        },
        usage={
            "prompt_tokens": openai_response.usage.prompt_tokens,
            "completion_tokens": openai_response.usage.completion_tokens,
            "total_tokens": openai_response.usage.total_tokens
        },
        latency_ms=latency_ms,
        client_ip=client_ip
    )
    
    logger.info(
        "chat_completion_success",
        request_id=request_id,
        tokens=openai_response.usage.total_tokens,
        latency_ms=latency_ms
    )
    
    # Print final OpenAI format response
    print(f"\n{'='*80}")
    print(f"✨ 返回OpenAI格式响应 - Request ID: {request_id}")
    print(f"{'='*80}")
    print(f"响应ID: {openai_response.id}")
    print(f"模型: {openai_response.model}")
    print(f"响应内容: {openai_response.choices[0].message.content[:500] if openai_response.choices[0].message.content else 'N/A'}{'...' if openai_response.choices[0].message.content and len(openai_response.choices[0].message.content) > 500 else ''}")
    print(f"完成原因: {openai_response.choices[0].finish_reason}")
    print(f"\nToken使用:")
    print(f"  输入tokens: {openai_response.usage.prompt_tokens}")
    print(f"  输出tokens: {openai_response.usage.completion_tokens}")
    print(f"  总tokens: {openai_response.usage.total_tokens}")
    print(f"总耗时: {latency_ms:.2f}ms")
    print(f"{'='*80}\n")
    
    return openai_response

async def _handle_streaming(
    request_id: str,
    request_data: ChatCompletionRequest,
    dashscope_messages: list,
    params: dict,
    client_ip: str,
    request_dict: dict,
    start_time: float
):
    """Handle streaming chat completion"""
    
    async def generate_stream():
        """Generate SSE stream"""
        chunk_id = generate_chat_completion_id()
        accumulated_content = ""
        
        try:
            # Get streaming generator
            stream_gen = await qwen_client.chat_completion(
                messages=dashscope_messages,
                **params
            )
            
            # First chunk with role
            first_chunk = converter.dashscope_to_openai_stream_chunk(
                qwen_chunk={"content": "", "finish_reason": None},
                model=request_data.model,
                chunk_id=chunk_id
            )
            first_chunk.choices[0].delta.role = "assistant"
            yield f"data: {first_chunk.model_dump_json()}\n\n"
            
            # Stream chunks
            async for qwen_chunk in stream_gen:
                content = qwen_chunk.get("content", "")
                accumulated_content += content
                
                openai_chunk = converter.dashscope_to_openai_stream_chunk(
                    qwen_chunk=qwen_chunk,
                    model=request_data.model,
                    chunk_id=chunk_id
                )
                
                yield f"data: {openai_chunk.model_dump_json()}\n\n"
            
            # Done
            yield "data: [DONE]\n\n"
            
            # Log completed stream
            latency_ms = (time.time() - start_time) * 1000
            
            # Print streaming completion info
            print(f"\n{'='*80}")
            print(f"✅ 流式响应完成 - Request ID: {request_id}")
            print(f"{'='*80}")
            print(f"累计内容长度: {len(accumulated_content)} 字符")
            print(f"内容预览: {accumulated_content[:200]}{'...' if len(accumulated_content) > 200 else ''}")
            print(f"总耗时: {latency_ms:.2f}ms")
            print(f"{'='*80}\n")
            
            logger_service.log_qwen_call(
                request_id=request_id,
                request=request_dict,
                response={
                    "content": accumulated_content,
                    "finish_reason": "stop"
                },
                latency_ms=latency_ms,
                client_ip=client_ip,
                stream=True
            )
            
        except Exception as e:
            error_msg = str(e)
            logger.error("streaming_error", request_id=request_id, error=error_msg)
            
            # Log error
            logger_service.log_qwen_call(
                request_id=request_id,
                request=request_dict,
                error=error_msg,
                client_ip=client_ip,
                stream=True
            )
    
    return StreamingResponse(
        generate_stream(),
        media_type="text/event-stream"
    )
