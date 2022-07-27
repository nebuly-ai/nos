import os.path as ops

import lightning as L

import nebuly.lightning as nebuly_lightning
from quick_start.components import PyTorchLightningScript, ImageServeGradio


@nebuly_lightning.optimized
class TrainDeploy(L.LightningFlow):
    def __init__(self):
        super().__init__()
        self.train_work = PyTorchLightningScript(
            script_path=ops.join(ops.dirname(__file__), "./train_script.py"),
            script_args=["--trainer.max_epochs=5"],
        )

        self.serve_work = ImageServeGradio(L.CloudCompute("cpu"))

    def run(self):
        # 1. Run the python script that trains the model
        self.train_work.run()

        # 2. when a checkpoint is available, deploy
        if self.train_work.best_model_path:
            self.serve_work.run(self.train_work.best_model_path)

    def configure_layout(self):
        tab_1 = {"name": "Model training", "content": self.train_work}
        tab_2 = {"name": "Interactive demo", "content": self.serve_work}
        return [tab_1, tab_2]


if __name__ == "__main__":
    obj = TrainDeploy()
    pass

app = L.LightningApp(TrainDeploy())
workflow = nebuly_lightning.LightningWorkflow(app)
