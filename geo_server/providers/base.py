import logging
from abc import ABC, abstractmethod
from typing import List, Dict, Any
from core.logger_config import setup_logger

class BaseProvider(ABC):
    def __init__(self, headless: bool = False, timeout: int = 30000):
        self.headless = headless
        self.timeout = timeout
        self.logger = setup_logger(self.__class__.__name__)

    @abstractmethod
    def search(self, keyword: str, prompt: str) -> Dict[str, Any]:
        """
        执行搜索并返回结果
        返回格式: {
            "full_text": str,
            "citations": List[Dict[str, str]] # [{"url": "...", "title": "..."}]
        }
        """
        pass
