from typing import List

from azureml.core import Workspace, ComputeTarget, RunConfiguration
from azureml.pipeline.core import Pipeline, PipelineStep
from azureml.pipeline.steps import PythonScriptStep

from common.nebuly.core import (
    WorkflowRunMetrics,
    OptimizationOptions,
    Workflow,
    TaskKind,
    Task,
    TaskOptimizer,
    OptimizationTarget,
    HardwareProvider,
)


class _AzureMachineLearningHardwareProvider(HardwareProvider):
    def __init__(self, workspace: Workspace):
        self._workspace = workspace

    def get_available_hardware(self) -> List[str]:
        for _, target in self._workspace.compute_targets.items():
            pass
        return []


class _AzureMachineLearningTask(Task):
    def __init__(
            self,
            workspace: Workspace,
            step: PythonScriptStep,
            kind: TaskKind,
            model_class,
            target: OptimizationTarget,
            name: str,
            display_name: str
    ):
        super().__init__(kind, model_class, target, name, display_name)
        self._step = step
        self._workspace = workspace

    def __get_compute_target(self, kinds: List[str]) -> ComputeTarget:
        return ComputeTarget(workspace=self._workspace, name="cpu-cluster")

    def set_hardware_kinds(self, kinds: List[str]):
        self._step._compute_target = self.__get_compute_target(kinds)

    def set_optimization_strategy(self, strategy: str):
        self._step._runconfig.environment_variables["NEBULLVM_CONFIG"] = strategy  # noqa


class TrainingPythonScriptStep(PythonScriptStep):
    def __init__(self, script_name: str, model_class, optimization_target: OptimizationTarget = None, **kwargs):
        super().__init__(script_name, **kwargs)
        self.kind = TaskKind.TRAINING
        self.model_class = model_class
        self.optimization_target = optimization_target


class InferencePythonScriptStep(PythonScriptStep):
    def __init__(self, script_name: str, model_class, optimization_target: OptimizationTarget = None, **kwargs):
        super().__init__(script_name, **kwargs)
        self.kind = TaskKind.INFERENCE
        self.model_class = model_class
        self.optimization_target = optimization_target


class TestPythonScriptStep(PythonScriptStep):
    def __init__(self, script_name: str, model_class, optimization_target: OptimizationTarget = None, **kwargs):
        super().__init__(script_name, **kwargs)
        self.kind = TaskKind.TEST
        self.model_class = model_class
        self.optimization_target = optimization_target


def _extract_tasks(workspace: Workspace, steps: List[PipelineStep]) -> List[Task]:
    task_steps = [
        s for s in steps
        if issubclass(s.__class__, (InferencePythonScriptStep, TrainingPythonScriptStep, TestPythonScriptStep))
    ]
    result = []
    for step in task_steps:
        task = _AzureMachineLearningTask(
            workspace=workspace,
            step=step,
            kind=step.kind,
            model_class=step.model_class,
            target=step.optimization_target,
            name=step.name,
            display_name=step.name,
        )
        result.append(task)
    return result


class OptimizedPipeline(Pipeline):
    def __init__(self, workspace, steps, **kwargs):
        super().__init__(workspace, steps, **kwargs)
        self.tasks = _extract_tasks(workspace, steps)
        self.optimizer = TaskOptimizer(_AzureMachineLearningHardwareProvider(workspace))

    def publish(self, *args, **kwargs):
        self.optimizer.optimize(self.tasks)
        super().publish(*args, **kwargs)


class AzureMachineLearningWorkflow(Workflow):
    def __int__(self, workspace: Workspace, pipeline: Pipeline):
        self._pipeline = pipeline
        self._workspace = workspace

    def publish(self):
        self._pipeline.publish()

    def run(self):
        self._pipeline.submit(experiment_name="test-run")

    def optimize(self, optimization_options: OptimizationOptions = None):
        pass

    def get_run_metrics(self) -> WorkflowRunMetrics:
        pass
