import warnings

warnings.simplefilter("ignore")
import logging
import os
from functools import partial
import torch
import torchvision.transforms as T
from quick_start.download import download_data
from subprocess import Popen
from lightning.app.storage import Path
from lightning.app.components.python import TracerPythonScript
from lightning.app.components.serve import ServeGradio
import gradio as gr
import nebuly.lightning as nebuly_lightning

logger = logging.getLogger(__name__)


class PyTorchLightningScript(TracerPythonScript, nebuly_lightning.TrainingTaskMixin):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, raise_exception=True, **kwargs)
        self.best_model_path = None
        self._process = None

    def configure_tracer(self):
        from pytorch_lightning import Trainer
        from pytorch_lightning.callbacks import Callback

        class CollectURL(Callback):
            def __init__(self, work):
                self._work = work

            def on_train_start(self, trainer, *_):
                self._work._process = Popen(
                    f"tensorboard --logdir='{trainer.logger.log_dir}' --host {self._work.host} --port {self._work.port}",
                    shell=True,
                )

        def trainer_pre_fn(self, *args, work=None, **kwargs):
            kwargs['callbacks'].append(CollectURL(work))
            return {}, args, kwargs

        tracer = super().configure_tracer()
        tracer.add_traced(Trainer, "__init__", pre_fn=partial(trainer_pre_fn, work=self))
        return tracer

    def run(self, *args, **kwargs):
        download_data("https://pl-flash-data.s3.amazonaws.com/assets_lightning/demo_weights.pt", "./")
        self.script_args += [
            "--trainer.limit_train_batches=4",
            "--trainer.limit_val_batches=4",
            "--trainer.callbacks=ModelCheckpoint",
            "--trainer.callbacks.monitor=val_acc",
        ]
        warnings.simplefilter("ignore")
        logger.info(f"Running train_script: {self.script_path}")
        super().run(*args, **kwargs)

    def on_after_run(self, res):
        lightning_module = res["cli"].trainer.lightning_module
        checkpoint = torch.load(res["cli"].trainer.checkpoint_callback.best_model_path)
        lightning_module.load_state_dict(checkpoint["state_dict"])
        lightning_module.to_torchscript("model_weight.pt")
        self.best_model_path = Path("model_weight.pt")


class ImageServeGradio(ServeGradio, nebuly_lightning.InferenceTaskMixin):
    inputs = gr.inputs.Image(type="pil", shape=(28, 28))
    outputs = gr.outputs.Label(num_top_classes=10)

    def __init__(self, cloud_compute, *args, **kwargs):
        super().__init__(*args, cloud_compute=cloud_compute, **kwargs)
        self.examples = None
        self.best_model_path = None
        self._transform = None
        self._labels = {idx: str(idx) for idx in range(10)}

    def run(self, best_model_path):
        download_data("https://pl-flash-data.s3.amazonaws.com/assets_lightning/images.tar.gz", "./")
        self.examples = [os.path.join(str("./images"), f) for f in os.listdir("./images")]
        self.best_model_path = best_model_path
        self._transform = T.Compose([T.Resize((28, 28)), T.ToTensor()])
        super().run()

    def predict(self, img):
        img = self._transform(img)[0]
        img = img.unsqueeze(0).unsqueeze(0)
        prediction = torch.exp(self.model(img))
        return {self._labels[i]: prediction[0][i].item() for i in range(10)}

    def build_model(self):
        model = torch.load(self.best_model_path)
        for p in model.parameters():
            p.requires_grad = False
        model.eval()
        return model
