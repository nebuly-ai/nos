import os

from azureml.core import Workspace, RunConfiguration
from azureml.pipeline.core import Pipeline, StepSequence

from nebuly.azure_ml import TrainingPythonScriptStep, TestPythonScriptStep, OptimizedPipeline


class PipelineBuilder:
    def __init__(self, ws: Workspace):
        self.workspace = ws

    def build_pipeline(self) -> Pipeline:
        training_step = TrainingPythonScriptStep(
            name="Training",
            model_class=None,
            script_name="__main__.py",
            source_directory=os.path.join("steps", "step-1", "src"),
            arguments=[
                "foo",
            ],
            allow_reuse=False,
        )
        test_step = TestPythonScriptStep(
            name="Test model",
            model_class=None,
            script_name="__main__.py",
            source_directory=os.path.join("steps", "step-2", "src"),
            arguments=[
                "bar",
            ],
            allow_reuse=False,
        )
        return OptimizedPipeline(workspace=self.workspace, steps=StepSequence([training_step, test_step]))


if __name__ == "__main__":
    workspace = Workspace.from_config()
    builder = PipelineBuilder(workspace)
    pipeline = builder.build_pipeline()
    pipeline.publish(
        name="Dummy pipeline",
        description="This is a dummy pipeline for test purposes"
    )
