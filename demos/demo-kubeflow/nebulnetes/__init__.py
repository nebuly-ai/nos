from typing import Callable, Optional

from kfp.dsl import BaseOp
from kubernetes.client.models import V1Affinity


def get_affinity(*args) -> V1Affinity:
    return V1Affinity()


def optimize(func: Callable, model: Optional = None):
    def _optimize_op(*args, **kwargs):
        operation: BaseOp = func(*args, **kwargs)

        affinity = get_affinity(model)
        operation.add_affinity(affinity)

        return operation

    return _optimize_op
