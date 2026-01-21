"""
geo_agent - OpenAI compatible agent service powered by qwen3-max

Main application entry point
"""
import uvicorn
from fastapi import FastAPI
from fastapi.responses import RedirectResponse

from app import __version__
from app.core.config import config
from app.core.logger import setup_logging, get_logger
from app.core.middleware import LoggingMiddleware, setup_cors
from app.api.health import router as health_router
from app.api.v1.chat import router as chat_router
from app.api.v1.models import router as models_router
from app.api.v1.completions import router as completions_router

# Setup logging
setup_logging(config.logging.level)
logger = get_logger(__name__)

# Create FastAPI app
app = FastAPI(
    title="geo_agent",
    description="OpenAI compatible agent service powered by qwen3-max",
    version=__version__,
    docs_url="/docs",
    redoc_url="/redoc"
)

# Setup CORS
setup_cors(app)

# Add logging middleware
app.add_middleware(LoggingMiddleware)

# Register routers
app.include_router(health_router, tags=["Health"])
app.include_router(chat_router, prefix="/v1", tags=["Chat"])
app.include_router(models_router, prefix="/v1", tags=["Models"])
app.include_router(completions_router, prefix="/v1", tags=["Completions"])

@app.get("/")
async def root():
    """Redirect to API docs"""
    return RedirectResponse(url="/docs")

@app.on_event("startup")
async def startup_event():
    """Startup event"""
    logger.info(
        "starting_geo_agent",
        version=__version__,
        host=config.server.host,
        port=config.server.port,
        log_level=config.logging.level
    )

@app.on_event("shutdown")
async def shutdown_event():
    """Shutdown event"""
    logger.info("shutting_down_geo_agent")

def main():
    """Run the application"""
    uvicorn.run(
        "main:app",
        host=config.server.host,
        port=config.server.port,
        reload=config.server.reload,
        log_level=config.logging.level.lower()
    )

if __name__ == "__main__":
    main()
