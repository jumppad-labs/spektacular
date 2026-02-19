"""Configuration models using Pydantic."""

import os
from pathlib import Path
from typing import Optional, Dict, Any
from pydantic import BaseModel, Field, validator
import yaml


class ApiConfig(BaseModel):
    """API configuration settings."""
    anthropic_api_key: str = Field(default="${ANTHROPIC_API_KEY}", description="Anthropic API key")
    timeout: int = Field(default=60, description="API timeout in seconds")


class ModelTiers(BaseModel):
    """Model tier definitions."""
    simple: str = Field(default="anthropic/claude-3-5-haiku-20241022", description="Simple complexity model")
    medium: str = Field(default="anthropic/claude-3-5-sonnet-20241022", description="Medium complexity model") 
    complex: str = Field(default="anthropic/claude-3-opus-20240229", description="Complex complexity model")


class ModelsConfig(BaseModel):
    """Model configuration settings."""
    default: str = Field(default="anthropic/claude-3-5-sonnet-20241022", description="Default model")
    tiers: ModelTiers = Field(default_factory=ModelTiers, description="Model tiers by complexity")


class ComplexityThresholds(BaseModel):
    """Complexity scoring thresholds."""
    simple: float = Field(default=0.3, ge=0.0, le=1.0, description="Simple complexity threshold")
    medium: float = Field(default=0.6, ge=0.0, le=1.0, description="Medium complexity threshold")
    complex: float = Field(default=0.8, ge=0.0, le=1.0, description="Complex complexity threshold")


class ComplexityConfig(BaseModel):
    """Complexity analysis configuration."""
    thresholds: ComplexityThresholds = Field(default_factory=ComplexityThresholds, description="Scoring thresholds")


class OutputConfig(BaseModel):
    """Output format configuration."""
    format: str = Field(default="markdown", description="Output format")
    include_metadata: bool = Field(default=True, description="Include metadata in output")


class SpektacularConfig(BaseModel):
    """Main Spektacular configuration."""
    api: ApiConfig = Field(default_factory=ApiConfig, description="API settings")
    models: ModelsConfig = Field(default_factory=ModelsConfig, description="Model configuration")
    complexity: ComplexityConfig = Field(default_factory=ComplexityConfig, description="Complexity analysis")
    output: OutputConfig = Field(default_factory=OutputConfig, description="Output settings")
    
    @classmethod
    def from_yaml_file(cls, config_path: Path) -> "SpektacularConfig":
        """Load configuration from YAML file."""
        if not config_path.exists():
            raise FileNotFoundError(f"Config file not found: {config_path}")
        
        with open(config_path, 'r') as f:
            data = yaml.safe_load(f)
        
        # Expand environment variables
        data = cls._expand_env_vars(data)
        return cls(**data)
    
    @classmethod
    def _expand_env_vars(cls, data: Any) -> Any:
        """Recursively expand environment variables in config data."""
        if isinstance(data, dict):
            return {key: cls._expand_env_vars(value) for key, value in data.items()}
        elif isinstance(data, list):
            return [cls._expand_env_vars(item) for item in data]
        elif isinstance(data, str) and data.startswith("${}") and data.endswith("}"):
            env_var = data[2:-1]
            return os.getenv(env_var, data)
        return data
    
    def to_yaml_file(self, config_path: Path) -> None:
        """Save configuration to YAML file."""
        config_path.parent.mkdir(parents=True, exist_ok=True)
        
        with open(config_path, 'w') as f:
            yaml.dump(
                self.dict(), 
                f, 
                default_flow_style=False, 
                indent=2,
                sort_keys=True
            )
    
    def get_model_for_complexity(self, complexity_score: float) -> str:
        """Get the appropriate model based on complexity score."""
        thresholds = self.complexity.thresholds
        
        if complexity_score < thresholds.simple:
            return self.models.tiers.simple
        elif complexity_score < thresholds.medium:
            return self.models.tiers.medium
        else:
            return self.models.tiers.complex


def save_default_config(spektacular_dir: Path) -> None:
    """Save default configuration to .spektacular directory."""
    config = SpektacularConfig()
    config_path = spektacular_dir / "config.yaml"
    config.to_yaml_file(config_path)
