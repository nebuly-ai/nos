import os

from azureml.core import Workspace
from azureml.pipeline.core import Pipeline, StepSequence
from azureml.pipeline.steps import PythonScriptStep


class PipelineBuilder:
    def __init__(self, workspace: Workspace):
        self.workspace = workspace

    def build_pipeline(self) -> Pipeline:
        training_step = PythonScriptStep(
            name="Training",
            script_name="__main__.py",
            source_directory=os.path.join("pipeline", "steps", "step-1", "src"),
            arguments=[
                "foo",
            ],
            allow_reuse=False,
        )
        test_step = PythonScriptStep(
            name="Test model",
            script_name="__main__.py",
            source_directory=os.path.join("pipeline", "steps", "step-2", "src"),
            arguments=[
                "bar",
            ],
            allow_reuse=False,
        )
        return Pipeline(workspace=self.workspace, steps=StepSequence([training_step, test_step]))


if __name__ == "__main__":
    workspace = Workspace.from_config()
    builder = PipelineBuilder(workspace)
    pipeline = builder.build_pipeline()
    pipeline.publish(
        name="Dummy pipeline",
        description="This is a dummy pipeline for test purposes"
    )
