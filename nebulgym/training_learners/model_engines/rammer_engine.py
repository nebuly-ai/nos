import json
import types
from typing import Dict

import torch.nn

from nebulgym.training_learners.model_engines.base import BaseEngine

try:
    from nnfusion.runner import PTRunner
except ImportError:
    import warnings

    warnings.warn(
        "No valid Rammer installation found. Using the Rammer backend "
        "will result in an error."
    )


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


class RammerEngine(BaseEngine):
    def __init__(self, model: torch.nn.Module):
        super(RammerEngine, self).__init__(model)
        rammer_dict = self._get_default_rammer_params()
        with CallMethodFixingEnvRunner(self._model):
            self._rammer_runner = PTRunner(model, rammer_dict)

    def run(self, *args, **kwargs):
        return self._rammer_runner.run_by_nnf(*args, **kwargs)

    def train(self, update_torch_model: bool = True):
        rammer_dict = self._get_default_rammer_params()
        if update_torch_model:
            self._model.train()
        with CallMethodFixingEnvRunner(self._model):
            self._rammer_runner = PTRunner(self._model, rammer_dict)

    def eval(self, update_torch_model: bool = True):
        if update_torch_model:
            self._model.eval()
        with CallMethodFixingEnvRunner(self._model):
            self._rammer_runner = PTRunner(self._model)

    @staticmethod
    def _get_default_rammer_params(use_default_trainer: bool = False) -> Dict:
        out_dict = {
            "autodiff": True,  # add backward graph
            "training_mode": True,  # move weight external
            "extern_result_memory": True,  # move result external
        }
        if use_default_trainer:
            out_dict.update(
                {
                    "training_optimizer": "'"
                    + json.dumps({"optimizer": "SGD", "learning_rate": 0.01})
                    + "'",  # training optimizer configs
                }
            )
        return out_dict
