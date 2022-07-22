import contextlib
import enum
import functools
from typing import Optional, Callable, List, Any

import kfp.compiler
import kfp.dsl as dsl

from .core import PipelineStep


class OperationKind(str, enum.Enum):
    TRAINING = "training"
    TEST = "test"
    DEPLOYMENT = "deployment"


class _PipelineOptimizer:
    def __init__(self):
        self.steps: List[PipelineStep] = []
        self._ops: List[dsl.ContainerOp] = []

    def register_op(
            self,
            operation: dsl.ContainerOp,
            operation_kind: OperationKind,
            model=None,
            optimization_target=None
    ):
        self._ops.append(operation)

    def optimize(self):
        for op in self._ops:
            print(op.human_name)


_optimizer: Optional[_PipelineOptimizer] = None


def optimize_test(operation: dsl.ContainerOp, model: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, OperationKind.TEST, model, optimization_target)


def optimize_deployment(operation: dsl.ContainerOp, model: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, OperationKind.DEPLOYMENT, model, optimization_target)


def optimize_training(operation: dsl.ContainerOp, model: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, OperationKind.TRAINING, model, optimization_target)

    # # TODO: infer affinity from Nebulex response
    # affinity = V1Affinity(
    #     node_affinity=V1NodeAffinity(preferred_during_scheduling_ignored_during_execution=[
    #         V1PreferredSchedulingTerm(
    #             preference=V1NodeSelectorTerm(
    #                 match_expressions=[
    #                     V1NodeSelectorRequirement(
    #                         key="foo",
    #                         operator="in",
    #                         values=["bar_1", "bar_2"],
    #                     ),
    #                 ]
    #             ),
    #             weight=10,
    #         )
    #     ])
    # )
    # # TODO: get nebulgym config from Nebulex
    # operation.container.add_env_variable(V1EnvVar(name="NEBULGYM_CONFIG", value=""))
    # # TODO: develop sidecar able to inject Nebulgym inside the training script of the operation
    # # operation.add_sidecar(Sidecar(
    # #     name="Nebulgym injector",
    # #     image="nebuly.ai/nebulgym-injector",
    # #     command="fooo",
    # # ))
    #
    # operation.add_affinity(affinity)
    # operation.add_pod_label("n8s.nebuly.ai/optimization-target", optimization_target)
    # operation.add_pod_label("n8s.nebuly.ai/optimized-for", "training")
    # operation.add_pod_label("n8s.nebuly.ai/sdk-version", "0.0.1")
    # operation.add_pod_label("n8s.nebuly.ai/nebulex-version", "v1")


def optimized_pipeline(pipeline_func: Callable):
    @functools.wraps(pipeline_func)
    def _wrap(*args, **kwargs) -> Any:
        res = pipeline_func(*args, **kwargs)
        _optimizer.optimize()
        return res

    return _wrap


@contextlib.contextmanager
def _use_optimizer(optimizer: _PipelineOptimizer):
    global _optimizer
    __previous_optimizer = _optimizer
    _optimizer = optimizer
    yield
    _optimizer = __previous_optimizer


class KfpPipelinePublisher:
    def __init__(
            self,
            compiler: kfp.compiler.Compiler = None,
    ):
        self._compiler = compiler
        self._optimizer = _PipelineOptimizer()
        if compiler is None:
            self._compiler = kfp.compiler.Compiler()

    def register_pipeline(self, pipeline_func: Callable, **kwargs):
        package_path = kwargs.get("package_path", None)
        if package_path is None:
            package_path = f"{pipeline_func._component_human_name}.yaml"  # noqa
        with _use_optimizer(self._optimizer):
            self._compiler.compile(pipeline_func, package_path, **kwargs)
