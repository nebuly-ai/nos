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
    KubeflowWorkflow,
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

    compiler = kfp.compiler.Compiler()
    package_path = f"{dummy_pipeline._component_human_name}.yaml"  # noqa
    compiler.compile(dummy_pipeline, package_path)

# Hook for registering the workflow to Nebuly
workflow = KubeflowWorkflow(None, dummy_pipeline)


### Nebuly portal ###
# # Case 1: user updates optimization target of a task from Nebuly's portal
# updated_workflow = manager.publish_optimized_workflow(
#     {
#         "task_1": {"target": "cost"}
#     }
# )
# update_workflow(updated_workflow)
#
# # Case 2: user wants to run a workflow
# manager.run_workflow(workflow)
#
# # Case 3: Nebuly's portal notices that a workflow can further be optimized. It notices that from:
# # 1) workflow metadata stored in DB
# # 2) run metrics from the manager (manager.get_run_metrics() -> Metrics)
#
#
# # Idea: before publishing a new version of a workflow, we run the new version and compare it with the previous one, so
# # that we can know if it's actually better. The comparison depends on the specific framework. For instance, in
# # a KubeFlow pipeline the comparison could check whether a certain step produces better metrics.
#
# # Note: workflow can potentially run for an undefined amount of time (think for instance to Lightning works)
