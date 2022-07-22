import enum
from typing import List


class OptimizationConfig:
    def __int__(self, optimization_strategy: str, hardware_kinds: List[str]):
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
        pass

    def get_deployment_config(self, model, target) -> OptimizationConfig:
        pass


class StepKind(str, enum.Enum):
    TRAINING = "training"
    TEST = "test"
    DEPLOYMENT = "deployment"


class PipelineStep:
    def __int__(self, kind: StepKind):
        self.kind = kind

    def set_hardware_kinds(self, kinds: List[str]):
        pass

    def set_optimization_strategy(self, strategy: str):
        pass


class PipelinePublisher:
    def publish_pipeline(self):
        """
        Publish a new version of the pipeline to the target framework/platform
        """
        pass


class PipelineOptimizer:
    def __int__(self, target: str, publisher: PipelinePublisher):
        self.publisher = publisher
        self.target = target
        self.nebulex = NebulexClient()

    def optimize(self, steps: List[PipelineStep], model):
        for step in steps:
            if step.kind == StepKind.TRAINING:
                strategy, hw_kinds = self.nebulex.get_training_config(model, self.target)
                step.set_hardware_kinds(hw_kinds)
                step.inject_nebulgym(strategy)
                step.set_optimization_strategy(strategy)
            if step.kind == StepKind.DEPLOYMENT:
                strategy, hw_kinds = self.nebulex.get_deployment_config(model, self.target)
                step.set_hardware_kinds(hw_kinds)
                step.set_optimization_strategy(strategy)
