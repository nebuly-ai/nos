import abc
import enum
from typing import List


class OptimizationConfig:
    def __init__(self, optimization_strategy: str, hardware_kinds: List[str]):
        """
        Defines how to perform an optimization in terms of optimization strategies and kinds of hardware required for
        applying those strategies.

        Parameters
        ----------
        optimization_strategy: str
            The optimization strategy to use for optimizing a model, either for training, test or deployment.
            In case of training, it corresponds to the Nebulgym configuration.
            In case of deployment, it corresponds to the Nebullvm configuration.
        hardware_kinds: List[str]
            The kinds of hardware required for performing the optimization.
        """
        self.optimization_strategy = optimization_strategy
        self.hardware_kinds = hardware_kinds


class NebulexClient:
    def get_training_config(self, model, target, available_hardware) -> OptimizationConfig:
        return OptimizationConfig("", [])

    def get_inference_config(self, model, target, available_hardware) -> OptimizationConfig:
        return OptimizationConfig("", [])

    def get_test_config(self, model, target, available_hardware) -> OptimizationConfig:
        return OptimizationConfig("", [])


class TaskKind(str, enum.Enum):
    TRAINING = "training"
    TEST = "test"
    INFERENCE = "inference"


class OptimizationTarget(str, enum.Enum):
    COST = "cost"
    TIME = "time"
    EMISSIONS = "emissions"


class Task(abc.ABC):
    def __init__(
            self,
            kind: TaskKind,
            model_class,
            target: OptimizationTarget,
            name: str,
            display_name: str
    ):
        self.kind = kind
        self.model_class = model_class
        self.target = target
        self.name = name
        self.display_name = display_name

    @abc.abstractmethod
    def set_hardware_kinds(self, kinds: List[str]):
        pass

    @abc.abstractmethod
    def set_optimization_strategy(self, strategy: str):
        pass


class HardwareProvider(abc.ABC):
    @abc.abstractmethod
    def get_available_hardware(self) -> List[str]:
        pass


class TaskOptimizer:
    def __init__(self, hardware_provider: HardwareProvider):
        self._hardware_provider = hardware_provider
        self._nebulex = NebulexClient()

    def optimize(self, tasks: List[Task]):
        available_hardware = self._hardware_provider.get_available_hardware()
        for task in tasks:
            config = None
            if task.kind == TaskKind.TRAINING:
                config = self._nebulex.get_training_config(task.model_class, task.target, available_hardware)
            if task.kind == TaskKind.INFERENCE:
                config = self._nebulex.get_inference_config(task.model_class, task.target, available_hardware)
            if task.kind == TaskKind.TEST:
                config = self._nebulex.get_test_config(task.model_class, task.target, available_hardware)
            if config is None:
                raise Exception(f"Task kind {task.kind} is not valid")
            task.set_optimization_strategy(config.optimization_strategy)
            task.set_hardware_kinds(config.hardware_kinds)


class WorkflowRunMetrics:
    pass


class OptimizationOptions:
    pass


class Workflow(abc.ABC):
    def __init__(self, name: str, tasks: List[Task], kind):
        self.name = name
        self.tasks = tasks
        self.kind = kind

    @abc.abstractmethod
    def publish(self):
        """Publish the managed workflow on the target platform (e.g. KubeFlow, Lightning, etc.)
        """
        pass

    @abc.abstractmethod
    def run(self):
        """Run the workflow on the target framework/platform"""
        pass

    @abc.abstractmethod
    def optimize(self, optimization_options: OptimizationOptions = None):
        """Optimize the workflow and update its tasks with the optimized configuration."""
        pass

    @abc.abstractmethod
    def get_run_metrics(self) -> WorkflowRunMetrics:
        """Return the aggregated metrics of all the runs of the workflow on the target framework/platform"""
        pass
