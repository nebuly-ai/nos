import torch.nn

from nebulgym.training_learners.model_engines.torch_engine import TorchEngine
from nebulgym.training_learners.model_engines.utils import (
    CallMethodFixingEnvRunner,
)

try:
    from torch_ort import ORTModule
except ImportError:
    import warnings

    warnings.warn(
        "No ONNXRuntime training library for pytorch has been detected. "
        "The ORT backend won't be used."
    )


class ORTEngine(TorchEngine):
    def __init__(self, model: torch.nn.Module):
        with CallMethodFixingEnvRunner(model):
            model = ORTModule(model)
        super().__init__(model)
