import pathlib

import kfp.compiler
from kfp import components
from kfp import dsl
from torch import nn

from nebulnetes.kubeflow import (
    optimize_training,
    optimize_inference,
    optimize_test,
    optimized_pipeline,
    KfpPipelinePublisher
)

KUBEFLOW_BASE_DIR = pathlib.Path("kubeflow")
PIPELINE_NAME = "dummy-pipeline"


class MyModel(nn.Module):
    def __init__(self):
        super(MyModel, self).__init__()
        self.flatten = nn.Flatten()
        self.linear_relu_stack = nn.Sequential(
            nn.Linear(28 * 28, 512),
            nn.ReLU(),
            nn.Linear(512, 512),
            nn.ReLU(),
            nn.Linear(512, 10)
        )

    def forward(self, x):
        x = self.flatten(x)
        logits = self.linear_relu_stack(x)
        return logits


def sleep(seconds: int):
    import time
    print(f"Sleeping for {seconds} seconds...")
    time.sleep(seconds)


def sleep_random():
    import time
    import random
    seconds = random.randint(0, 10)
    print(f"Sleeping for {seconds} seconds...")
    time.sleep(seconds)


sleep_op = components.create_component_from_func(sleep)
sleep_random_op = components.create_component_from_func(sleep_random)


@optimized_pipeline
@dsl.pipeline(
    name=PIPELINE_NAME,
    description='A toy pipeline that does nothing.'
)
def dummy_pipeline(seconds: int):
    random_sleep_1_task = sleep_random_op()
    optimize_training(random_sleep_1_task, model_class=MyModel, optimization_target=None)

    sleep_task = sleep_op(seconds).after(random_sleep_1_task)
    optimize_test(sleep_task, model_class=MyModel, optimization_target=None)

    random_sleep_2_task = sleep_random_op().after(sleep_task)
    optimize_inference(random_sleep_2_task, model_class=MyModel, optimization_target=None)

    return True


if __name__ == "__main__":
    # client = kfp.Client(host="http://localhost:8081")

    pipeline_output_path = KUBEFLOW_BASE_DIR / pathlib.Path("dummy-pipeline.yaml")
    compiler = kfp.compiler.Compiler()
    publisher = KfpPipelinePublisher(compiler)
    publisher.register_pipeline(dummy_pipeline)

    #
    # try:
    #     res = client.upload_pipeline(
    #         pipeline_name=PIPELINE_NAME,
    #         pipeline_package_path=str(pipeline_output_path),
    #     )
    # except Exception:
    #     client.upload_pipeline_version(
    #         pipeline_name=PIPELINE_NAME,
    #         pipeline_package_path=str(pipeline_output_path),
    #         pipeline_version_name=str(uuid.uuid4()),
    #     )
