"""
OpenAI API compatible data models
Based on: https://platform.openai.com/docs/api-reference/chat
"""
from typing import List, Optional, Dict, Any, Literal, Union
from pydantic import BaseModel, Field

# ============ Request Models ============

class ChatMessage(BaseModel):
    """Chat message"""
    role: Literal["system", "user", "assistant", "function"]
    content: str
    name: Optional[str] = None
    function_call: Optional[Dict[str, Any]] = None

class ChatCompletionRequest(BaseModel):
    """Chat completion request"""
    model: str
    messages: List[ChatMessage]
    temperature: Optional[float] = Field(default=0.7, ge=0, le=2)
    top_p: Optional[float] = Field(default=1.0, ge=0, le=1)
    n: Optional[int] = Field(default=1, ge=1)
    stream: Optional[bool] = False
    stop: Optional[Union[str, List[str]]] = None
    max_tokens: Optional[int] = Field(default=2000, ge=1)
    presence_penalty: Optional[float] = Field(default=0, ge=-2, le=2)
    frequency_penalty: Optional[float] = Field(default=0, ge=-2, le=2)
    logit_bias: Optional[Dict[str, float]] = None
    user: Optional[str] = None
    
    class Config:
        json_schema_extra = {
            "example": {
                "model": "qwen3-max",
                "messages": [
                    {"role": "system", "content": "You are a helpful assistant."},
                    {"role": "user", "content": "Hello!"}
                ],
                "temperature": 0.7,
                "max_tokens": 2000
            }
        }

class CompletionRequest(BaseModel):
    """Text completion request"""
    model: str
    prompt: Union[str, List[str]]
    temperature: Optional[float] = Field(default=0.7, ge=0, le=2)
    top_p: Optional[float] = Field(default=1.0, ge=0, le=1)
    n: Optional[int] = Field(default=1, ge=1)
    stream: Optional[bool] = False
    stop: Optional[Union[str, List[str]]] = None
    max_tokens: Optional[int] = Field(default=2000, ge=1)
    presence_penalty: Optional[float] = Field(default=0, ge=-2, le=2)
    frequency_penalty: Optional[float] = Field(default=0, ge=-2, le=2)
    user: Optional[str] = None

# ============ Response Models ============

class Usage(BaseModel):
    """Token usage statistics"""
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int

class ChatChoice(BaseModel):
    """Chat completion choice"""
    index: int
    message: ChatMessage
    finish_reason: Optional[str] = None

class CompletionChoice(BaseModel):
    """Text completion choice"""
    index: int
    text: str
    finish_reason: Optional[str] = None

class ChatCompletionResponse(BaseModel):
    """Chat completion response"""
    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: List[ChatChoice]
    usage: Usage
    
    class Config:
        json_schema_extra = {
            "example": {
                "id": "chatcmpl-abc123",
                "object": "chat.completion",
                "created": 1677652288,
                "model": "qwen3-max",
                "choices": [{
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": "Hello! How can I help you today?"
                    },
                    "finish_reason": "stop"
                }],
                "usage": {
                    "prompt_tokens": 20,
                    "completion_tokens": 10,
                    "total_tokens": 30
                }
            }
        }

class CompletionResponse(BaseModel):
    """Text completion response"""
    id: str
    object: str = "text_completion"
    created: int
    model: str
    choices: List[CompletionChoice]
    usage: Usage

# ============ Streaming Models ============

class ChatCompletionChunkDelta(BaseModel):
    """Delta content in streaming"""
    role: Optional[str] = None
    content: Optional[str] = None

class ChatCompletionChunkChoice(BaseModel):
    """Streaming choice"""
    index: int
    delta: ChatCompletionChunkDelta
    finish_reason: Optional[str] = None

class ChatCompletionChunk(BaseModel):
    """Streaming chunk"""
    id: str
    object: str = "chat.completion.chunk"
    created: int
    model: str
    choices: List[ChatCompletionChunkChoice]

# ============ Models List ============

class Model(BaseModel):
    """Model information"""
    id: str
    object: str = "model"
    created: int
    owned_by: str

class ModelList(BaseModel):
    """List of available models"""
    object: str = "list"
    data: List[Model]

# ============ Error Models ============

class ErrorDetail(BaseModel):
    """Error detail"""
    message: str
    type: str
    param: Optional[str] = None
    code: Optional[str] = None

class ErrorResponse(BaseModel):
    """Error response"""
    error: ErrorDetail
