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
    def get_training_config(self, model, target) -> OptimizationConfig:
        return OptimizationConfig("", [])

    def get_inference_config(self, model, target) -> OptimizationConfig:
        return OptimizationConfig("", [])

    def get_test_config(self, model, target) -> OptimizationConfig:
        return OptimizationConfig("", [])


class TaskKind(str, enum.Enum):
    TRAINING = "training"
    TEST = "test"
    INFERENCE = "deployment"


class OptimizationTarget(str, enum.Enum):
    COST = "cost"
    TIME = "time"
    EMISSIONS = "emissions"


class Task(abc.ABC):
    def __init__(self, kind: TaskKind, model_class, target: OptimizationTarget):
        self.kind = kind
        self.model_class = model_class
        self.target = target

    @abc.abstractmethod
    def set_hardware_kinds(self, kinds: List[str]):
        pass

    @abc.abstractmethod
    def set_optimization_strategy(self, strategy: str):
        pass


class PipelinePublisher:
    def publish_pipeline(self):
        """
        Publish a new version of the pipeline to the target framework/platform
        """
        pass


class TaskOptimizer:
    def __init__(self):
        self._nebulex = NebulexClient()

    def optimize(self, tasks: List[Task]):
        for task in tasks:
            config = None
            if task.kind == TaskKind.TRAINING:
                config = self._nebulex.get_training_config(task.model_class, task.target)
            if task.kind == TaskKind.INFERENCE:
                config = self._nebulex.get_inference_config(task.model_class, task.target)
            if task.kind == TaskKind.TEST:
                config = self._nebulex.get_test_config(task.model_class, task.target)
            if config is None:
                raise Exception(f"Task kind {task.kind} is not valid")
            task.set_optimization_strategy(config.optimization_strategy)
            task.set_hardware_kinds(config.hardware_kinds)
