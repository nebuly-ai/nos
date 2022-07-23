import contextlib
import functools
from typing import Optional, Callable, List, Any

import kfp.compiler
import kfp.dsl as dsl
from kfp.dsl import Sidecar
from kubernetes.client import (
    V1Affinity,
    V1NodeAffinity,
    V1PreferredSchedulingTerm,
    V1NodeSelectorTerm,
    V1NodeSelectorRequirement, V1EnvVar,
)

from . import core


class _KubeFlowTask(core.Task):
    def __init__(
            self,
            kind: core.TaskKind,
            target: core.OptimizationTarget,
            op: dsl.ContainerOp,
            model_class=None
    ):
        super().__init__(kind, target, model_class)
        self._op = op
        self._op.add_pod_label("n8s.nebuly.ai/optimization-target", target)
        self._op.add_pod_label("n8s.nebuly.ai/sdk-version", "0.0.1")
        self._op.add_pod_label("n8s.nebuly.ai/nebulex-version", "v1")

    def set_hardware_kinds(self, kinds: List[str]):
        affinity = V1Affinity(
            node_affinity=V1NodeAffinity(preferred_during_scheduling_ignored_during_execution=[
                V1PreferredSchedulingTerm(
                    preference=V1NodeSelectorTerm(
                        match_expressions=[
                            V1NodeSelectorRequirement(
                                key="foo",
                                operator="in",
                                values=["bar_1", "bar_2"],
                            ),
                        ]
                    ),
                    weight=10,
                )
            ])
        )
        self._op.add_affinity(affinity)

    def set_optimization_strategy(self, strategy: str):
        if self.kind == core.TaskKind.TRAINING:
            self._op.container.add_env_variable(V1EnvVar(name="NEBULGYM_CONFIG", value=strategy))
            self._op.add_sidecar(Sidecar(
                name="Nebulgym injector",
                image="nebuly.ai/nebulgym-injector",
                command="somehow inject nebulgym :)",
            ))
        if self.kind == core.TaskKind.INFERENCE:
            self._op.container.add_env_variable(V1EnvVar(name="NEBULLVM_CONFIG", value=strategy))
            self._op.add_sidecar(Sidecar(
                name="Nebullvm injector",
                image="nebuly.ai/nebullvm-injector",
                command="somehow inject nebullvm :)",
            ))
        self._op.add_pod_label("n8s.nebuly.ai/optimized-for", self.kind.value)


class _PipelineOptimizer:
    def __init__(self):
        self._tasks: List[core.Task] = []
        self._optimizer = core.TaskOptimizer()

    def register_op(
            self,
            operation: dsl.ContainerOp,
            kind: core.TaskKind,
            model=None,
            optimization_target=None
    ):
        task = _KubeFlowTask(kind, optimization_target, operation, model)
        self._tasks.append(task)

    def optimize(self):
        self._optimizer.optimize(self._tasks)


_optimizer: Optional[_PipelineOptimizer] = None


def optimize_test(operation: dsl.ContainerOp, model_class: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, core.TaskKind.TEST, model_class, optimization_target)


def optimize_inference(operation: dsl.ContainerOp, model_class: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, core.TaskKind.INFERENCE, model_class, optimization_target)


def optimize_training(operation: dsl.ContainerOp, model_class: Optional = None, optimization_target: Optional = None):
    if _optimizer is not None:
        _optimizer.register_op(operation, core.TaskKind.TRAINING, model_class, optimization_target)


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
