"""
OpenAI /v1/models endpoint
"""
from fastapi import APIRouter
from app.models.openai import ModelList, Model
from app.utils.helpers import get_current_timestamp

router = APIRouter()

@router.get("/models", response_model=ModelList)
async def list_models():
    """
    List available models (OpenAI compatible)
    
    Returns list of supported models
    """
    return ModelList(
        object="list",
        data=[
            Model(
                id="qwen3-max",
                object="model",
                created=get_current_timestamp(),
                owned_by="alibaba-cloud"
            ),
            Model(
                id="qwen-max",
                object="model",
                created=get_current_timestamp(),
                owned_by="alibaba-cloud"
            )
        ]
    )
