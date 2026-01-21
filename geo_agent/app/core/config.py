"""
Configuration management
"""
import os
import yaml
from typing import Optional, List
from pathlib import Path
from pydantic import Field
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

class ServerConfig(BaseSettings):
    """Server configuration"""
    host: str = Field(default="0.0.0.0")
    port: int = Field(default=8100)
    workers: int = Field(default=4)
    reload: bool = Field(default=False)

class QwenConfig(BaseSettings):
    """Qwen API configuration"""
    api_key: str = Field(default="")
    model: str = Field(default="qwen-max")
    timeout: int = Field(default=60)
    max_retries: int = Field(default=3)

class LoggingConfig(BaseSettings):
    """Logging configuration"""
    level: str = Field(default="INFO")
    format: str = Field(default="json")
    rotation: str = Field(default="daily")
    retention: str = Field(default="30d")

class SecurityConfig(BaseSettings):
    """Security configuration"""
    api_keys: Optional[str] = Field(default=None)
    
    def get_api_keys(self) -> List[str]:
        """Parse API keys from comma-separated string"""
        if not self.api_keys:
            return []
        return [k.strip() for k in self.api_keys.split(",") if k.strip()]

class Config:
    """Application configuration"""
    
    def __init__(self):
        self.server = ServerConfig()
        self.qwen = QwenConfig()
        self.logging = LoggingConfig()
        self.security = SecurityConfig()
        
        # Load from config.yaml if exists
        config_path = Path(__file__).parent.parent.parent / "config.yaml"
        if config_path.exists():
            self._load_yaml_config(config_path)
        
        # Override with environment variables
        self._load_env_config()
    
    def _load_yaml_config(self, config_path: Path):
        """Load configuration from YAML file"""
        with open(config_path, 'r', encoding='utf-8') as f:
            yaml_config = yaml.safe_load(f)
            
            if yaml_config:
                # Replace environment variables in YAML
                yaml_config = self._replace_env_vars(yaml_config)
                
                # Update configurations
                if 'server' in yaml_config:
                    self.server = ServerConfig(**yaml_config['server'])
                if 'qwen' in yaml_config:
                    self.qwen = QwenConfig(**yaml_config['qwen'])
                if 'logging' in yaml_config:
                    self.logging = LoggingConfig(**yaml_config['logging'])
                if 'security' in yaml_config:
                    self.security = SecurityConfig(**yaml_config['security'])
    
    def _replace_env_vars(self, obj):
        """Replace ${VAR} with environment variables"""
        if isinstance(obj, dict):
            return {k: self._replace_env_vars(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [self._replace_env_vars(item) for item in obj]
        elif isinstance(obj, str) and obj.startswith('${') and obj.endswith('}'):
            var_name = obj[2:-1]
            return os.getenv(var_name, '')
        return obj
    
    def _load_env_config(self):
        """Override with environment variables"""
        if os.getenv('DASHSCOPE_API_KEY'):
            self.qwen.api_key = os.getenv('DASHSCOPE_API_KEY')
        if os.getenv('PORT'):
            self.server.port = int(os.getenv('PORT'))
        if os.getenv('HOST'):
            self.server.host = os.getenv('HOST')
        if os.getenv('LOG_LEVEL'):
            self.logging.level = os.getenv('LOG_LEVEL')
        if os.getenv('AGENT_API_KEYS'):
            self.security.api_keys = os.getenv('AGENT_API_KEYS')
        if os.getenv('QWEN_MODEL'):
            self.qwen.model = os.getenv('QWEN_MODEL')

# Global configuration instance
config = Config()
