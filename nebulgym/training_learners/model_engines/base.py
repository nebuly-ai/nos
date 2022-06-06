from abc import ABC, abstractmethod

import torch.nn


class BaseEngine(ABC):
    """Base class for describing a DL Engine.

    Attributes:
        model (Module): Pytorch model.
    """

    def __init__(self, model: torch.nn.Module):
        self._model = model

    def __call__(self, *args, **kwargs):
        return self.run(*args, **kwargs)

    def forward(self, *args, **kwargs):
        return self.run(*args, **kwargs)

    def predict(self, *args, **kwargs):
        return self.run(*args, **kwargs)

    @abstractmethod
    def run(self, *args, **kwargs):
        raise NotImplementedError

    @abstractmethod
    def train(self, update_torch_model: bool = True):
        raise NotImplementedError

    @abstractmethod
    def eval(self, update_torch_model: bool = True):
        raise NotImplementedError

    @property
    def original_model(self):
        return self._model
