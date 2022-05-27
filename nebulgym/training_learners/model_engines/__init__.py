from typing import Dict, Type

from nebulgym.training_learners.model_engines.base import BaseEngine
from nebulgym.training_learners.model_engines.torch_engine import TorchEngine
from nebulgym.base import NebulgymBackend

ENGINE_MAP: Dict[NebulgymBackend, Type[BaseEngine]] = {
    NebulgymBackend.PYTORCH: TorchEngine,
}
