import pathlib

import kfp.compiler
from kfp import components
from kfp import dsl

from nebulnetes.kubeflow import (
    optimize_training,
    optimize_deployment,
    optimize_test,
    optimized_pipeline,
    KfpPipelinePublisher
)

KUBEFLOW_BASE_DIR = pathlib.Path("kubeflow")
PIPELINE_NAME = "dummy-pipeline"


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
    optimize_training(random_sleep_1_task, model=None, optimization_target=None)

    sleep_task = sleep_op(seconds).after(random_sleep_1_task)
    optimize_test(sleep_task, model=None, optimization_target=None)

    random_sleep_2_task = sleep_random_op().after(sleep_task)
    optimize_deployment(random_sleep_2_task, model=None, optimization_target=None)

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
