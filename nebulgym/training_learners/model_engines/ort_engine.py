import torch.nn

from nebulgym.decorators.helpers import suppress_warnings, suppress_stdout
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
    """ONNXRuntime engine."""

    def __init__(self, model: torch.nn.Module):
        with CallMethodFixingEnvRunner(model):
            ort_model = ORTModule(model)
        super().__init__(ort_model)
        self._original_model = model

    @suppress_stdout
    @suppress_warnings
    def run(self, *args, **kwargs):
        with CallMethodFixingEnvRunner(self._original_model):
            return self._model.forward(*args, **kwargs)

    @property
    def original_model(self):
        return self._original_model
