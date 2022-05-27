import torch.nn

from nebulgym.training_learners.model_engines.base import BaseEngine


class TorchEngine(BaseEngine):
    def __init__(self, model: torch.nn.Module):
        super().__init__(model)

    def run(self, *args, **kwargs):
        return self._model.forward(*args, **kwargs)

    def eval(self, update_torch_model: bool = True):
        if update_torch_model:
            self._model.eval()
        return self

    def train(self, update_torch_model: bool = True):
        if update_torch_model:
            self._model.train()
        return self
