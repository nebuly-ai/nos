import pathlib
import sys
import tempfile

import typer
from loguru import logger

from optimizer import services
from optimizer.models import StorageKind, OptimizationTarget, ModelDescriptor

ARG_SOURCE_MODEL_URI_HELP = "URI pointing to the model to optimize"
ARG_MODEL_LIBRARY_BASE_URI_HELP = "URI pointing to the space of the model library dedicated to the current model " \
                                  "optimization. The optimized model will be uploaded here."
ARG_OPTIMIZED_MODEL_DESCRIPTOR_URI_HELP = "URI pointing to the path where to upload the json file describing the " \
                                          "optimized model "
ARG_TARGET_HELP = "The target for which the model will be optimized"


def setup_logging(debug=False):
    log_level = "DEBUG" if debug is True else "INFO"
    log_format = "[<green>{time}</green> - <level>{level:<8}</level>] " \
                 "<cyan>{module}</cyan>:<cyan>{function}</cyan>:<cyan>{line}</cyan> " \
                 "- {message}"
    log_config = {
        "handlers": [
            {
                "sink": sys.stdout,
                "format": log_format,
                "colorize": True,
                "level": log_level,
            },
        ]
    }
    logger.configure(**log_config)


class Runner:
    def __init__(
            self,
            optimizer: services.ModelOptimizer,
            model_library: services.ModelLibrary,
            hardware_selector: services.HardwareSelector,
    ):
        self.optimizer = optimizer
        self.model_library = model_library
        self.hardware_selector = hardware_selector

    def run(self, source_model_uri):
        # Run optimization
        optimized_model_path = self.optimizer.optimize_model(source_model_uri)
        # Upload optimized model to model library
        optimized_model_uri = self.model_library.upload_model(optimized_model_path)
        # Select hardware for optimized model
        selected_hardware = self.hardware_selector.select_hardware_for_model(optimized_model_path)
        # Build and upload optimized model descriptor
        optimized_model_descriptor = ModelDescriptor(optimized_model_uri, selected_hardware)
        self.model_library.upload_model_descriptor(optimized_model_descriptor)


def cli(
        source_model_uri: str = typer.Argument(..., help=ARG_SOURCE_MODEL_URI_HELP),
        model_library_base_uri: str = typer.Argument(..., help=ARG_MODEL_LIBRARY_BASE_URI_HELP),
        optimized_model_descriptor_uri: str = typer.Argument(..., help=ARG_OPTIMIZED_MODEL_DESCRIPTOR_URI_HELP),
        model_library_storage_kind: StorageKind = typer.Argument(..., case_sensitive=False),
        target: OptimizationTarget = typer.Argument(..., case_sensitive=False, help=ARG_TARGET_HELP),
        debug: bool = False
):
    setup_logging(debug)

    with tempfile.TemporaryDirectory() as t:
        # Init services
        temp_dir = pathlib.Path(t)
        optimizer = services.ModelOptimizer(temp_dir)
        model_library = services.ModelLibraryFactory.new_model_library(
            model_library_storage_kind,
            model_library_base_uri,
            optimized_model_descriptor_uri
        )
        hardware_selector = services.HardwareSelector(target)
        runner = Runner(optimizer, model_library, hardware_selector)
        # Run optimization
        runner.run(source_model_uri)
