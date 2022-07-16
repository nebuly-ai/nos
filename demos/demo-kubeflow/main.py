import pathlib
import uuid

import kfp.compiler

from pipelines import mnist_pipeline

KUBEFLOW_BASE_DIR = pathlib.Path("kubeflow")

if __name__ == "__main__":
    client = kfp.Client(host="http://localhost:8081")

    # Create a new pipeline version of "Sleepy pipeline"
    pipeline_output_path = KUBEFLOW_BASE_DIR / pathlib.Path("mnist_pipeline.yaml")
    kfp.compiler.Compiler().compile(
        pipeline_func=mnist_pipeline.build_pipeline,
        package_path=str(pipeline_output_path),
    )
    res = client.upload_pipeline(
        pipeline_package_path=str(pipeline_output_path),
    )

    # List pipelines
    resp = client.list_pipelines()
    for p in resp.pipelines:
        print(f"Pipeline: {p.name}, URL: {p.url}")
