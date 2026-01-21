"""
OpenAI /v1/completions endpoint (optional, for backward compatibility)
"""
from fastapi import APIRouter, HTTPException
from app.models.openai import CompletionRequest, CompletionResponse

router = APIRouter()

@router.post("/completions", response_model=CompletionResponse)
async def create_completion(request: CompletionRequest):
    """
    Text completion endpoint (legacy)
    
    Note: This endpoint is optional and can be implemented later if needed.
    Most modern applications use /v1/chat/completions instead.
    """
    raise HTTPException(
        status_code=501,
        detail="Text completions not yet implemented. Please use /v1/chat/completions instead."
    )
