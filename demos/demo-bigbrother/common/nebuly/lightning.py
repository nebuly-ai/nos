import typing
from typing import List, Optional

import lightning
import lightning as L
from lightning_cloud.openapi.api import lightningapp_v2_service_api as app_apis
from lightning_app.core import LightningWork, LightningApp

from . import core
from .core import WorkflowRunMetrics, OptimizationOptions


class _LightningTask(core.Task):
    def __init__(
            self,
            work: LightningWork,
            kind: core.TaskKind,
            model_class,
            target: core.OptimizationTarget,
    ):
        self.work = work
        super().__init__(kind, model_class, target, work.name, work.name)

    @staticmethod
    def _get_lightning_cloud_compute(kinds: List[str]) -> lightning.CloudCompute:
        """Return the most suitable Lightning cloud compute according to the hardware kinds provided as argument"""
        return lightning.CloudCompute(name="cpu", preemptible=True)

    def set_hardware_kinds(self, kinds: List[str]):
        compute = self._get_lightning_cloud_compute(kinds)
        self.work.cloud_compute = compute

    def set_optimization_strategy(self, strategy: str):
        # TODO (env variable? add trace for injecting nebulgym/nebullvm? anything else?)
        pass


class TaskMixin:
    pass


class TrainingTaskMixin(TaskMixin):
    kind: core.TaskKind = core.TaskKind.TRAINING
    model_class: Optional[typing.Type] = None
    optimization_target: core.OptimizationTarget = core.OptimizationTarget.TIME


class InferenceTaskMixin(TaskMixin):
    kind: core.TaskKind = core.TaskKind.INFERENCE
    model_class: Optional[typing.Type] = None
    optimization_target: core.OptimizationTarget = core.OptimizationTarget.TIME


def _extract_tasks(works: List[lightning.LightningWork]) -> List[core.Task]:
    return [
        _LightningTask(
            work,
            kind=work.kind,
            model_class=work.model_class,
            target=work.optimization_target,
        ) for work in works
        if issubclass(work.__class__, TaskMixin)
    ]


def optimized(cls):
    original_init = cls.__init__

    def optimize(flow: L.LightningFlow):
        tasks = _extract_tasks(flow.works())
        core.TaskOptimizer().optimize(tasks)

    def new_init(self, *args, **kwargs):
        original_init(self, *args, **kwargs)
        optimize(self)

    cls.__init__ = new_init
    return cls


class LightningWorkflow(core.Workflow):
    KIND = "Lightning"

    def __init__(self, app: LightningApp):
        tasks = _extract_tasks(app.works)
        super().__init__(app.root.name, tasks, self.KIND)
        self.app = app
        self._optimizer = core.TaskOptimizer()
        self.lightning_cloud_service = app_apis.LightningappV2ServiceApi()

    def publish(self):
        # self.lightning_cloud_service.do_something()
        pass

    def run(self):
        # WFT do we do?
        # self.lightning_cloud_service.do_something()
        pass

    def optimize(self, optimization_options: OptimizationOptions = None):
        self._optimizer.optimize(self.tasks)

    def get_run_metrics(self) -> WorkflowRunMetrics:
        pass
