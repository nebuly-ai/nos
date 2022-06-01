import json
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


class RammerEngine(BaseEngine):
    def __init__(self, model: torch.nn.Module):
        super(RammerEngine, self).__init__(model)
        rammer_dict = self._get_default_rammer_params()
        self._rammer_runner = PTRunner(model, rammer_dict)

    def run(self, *args, **kwargs):
        return self._rammer_runner.run_by_nnf(*args, **kwargs)

    def train(self, update_torch_model: bool = True):
        rammer_dict = self._get_default_rammer_params()
        if update_torch_model:
            self._model.train()
        self._rammer_runner = PTRunner(self._model, rammer_dict)

    def eval(self, update_torch_model: bool = True):
        if update_torch_model:
            self._model.eval()
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
