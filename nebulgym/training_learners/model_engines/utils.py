import types

import torch.nn


class CallMethodFixingEnvRunner:
    def __init__(self, model: torch.nn.Module):
        self._prev_call = None
        self._model = model

    def __enter__(self):
        self._prev_call = self._model.__call__
        self._model.__call__ = types.MethodType(
            torch.nn.Module.__call__, self._model
        )

    def __exit__(self, exc_type, exc_val, exc_tb):
        self._model.__call__ = self._prev_call
