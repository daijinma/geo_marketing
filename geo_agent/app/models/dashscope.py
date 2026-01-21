"""
DashScope API data models for qwen
Reference: https://help.aliyun.com/zh/dashscope/developer-reference/api-details
"""
from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field

# ============ Request Models ============

class DashScopeMessage(BaseModel):
    """DashScope message format"""
    role: str  # system, user, assistant
    content: str

class DashScopeInput(BaseModel):
    """DashScope input"""
    messages: List[DashScopeMessage]

class DashScopeParameters(BaseModel):
    """DashScope parameters"""
    result_format: str = "message"
    temperature: Optional[float] = None
    top_p: Optional[float] = None
    top_k: Optional[int] = None
    max_tokens: Optional[int] = None
    stop: Optional[List[str]] = None
    enable_search: Optional[bool] = False
    incremental_output: Optional[bool] = False
    seed: Optional[int] = None

class DashScopeRequest(BaseModel):
    """DashScope API request"""
    model: str
    input: DashScopeInput
    parameters: Optional[DashScopeParameters] = None

# ============ Response Models ============

class DashScopeUsage(BaseModel):
    """Token usage"""
    input_tokens: int = Field(default=0)
    output_tokens: int = Field(default=0)
    total_tokens: int = Field(default=0)

class DashScopeOutputMessage(BaseModel):
    """Output message"""
    role: str
    content: str

class DashScopeOutput(BaseModel):
    """Response output"""
    text: Optional[str] = None
    finish_reason: Optional[str] = None
    choices: Optional[List[Dict[str, Any]]] = None

class DashScopeResponse(BaseModel):
    """DashScope API response"""
    status_code: int = Field(default=200)
    request_id: str
    code: Optional[str] = None
    message: Optional[str] = None
    output: Optional[DashScopeOutput] = None
    usage: Optional[DashScopeUsage] = None
