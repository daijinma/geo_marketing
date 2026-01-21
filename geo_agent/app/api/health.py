"""
Health check endpoint
"""
from fastapi import APIRouter
from pydantic import BaseModel

from app import __version__
from app.services.qwen_client import qwen_client

router = APIRouter()

class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    version: str
    qwen_api_configured: bool

@router.get("/health", response_model=HealthResponse)
async def health_check():
    """
    Health check endpoint
    
    Returns service status and configuration info
    """
    return HealthResponse(
        status="healthy",
        version=__version__,
        qwen_api_configured=qwen_client.validate_api_key()
    )
