"""
Utility helper functions
"""
import time
import uuid
from typing import Optional

def generate_request_id(prefix: str = "req") -> str:
    """Generate a unique request ID"""
    return f"{prefix}_{uuid.uuid4().hex[:16]}"

def generate_chat_completion_id() -> str:
    """Generate OpenAI-style chat completion ID"""
    return f"chatcmpl-{uuid.uuid4().hex[:16]}"

def generate_completion_id() -> str:
    """Generate OpenAI-style completion ID"""
    return f"cmpl-{uuid.uuid4().hex[:16]}"

def get_current_timestamp() -> int:
    """Get current Unix timestamp"""
    return int(time.time())

def safe_get(data: dict, *keys, default=None):
    """Safely get nested dict value"""
    for key in keys:
        if isinstance(data, dict):
            data = data.get(key)
            if data is None:
                return default
        else:
            return default
    return data if data is not None else default
