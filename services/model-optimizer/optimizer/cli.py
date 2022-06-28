import enum
import time

import typer
from loguru import logger
import sys

ARG_SOURCE_MODEL_URI_HELP = "URI pointing to the model to optimize"
ARG_DESTINATION_BASE_URI_HELP = "URI pointing to the destination to which upload the optimized model and any other " \
                                "artifact "
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


class OptimizationTarget(str, enum.Enum):
    cost = "cost"
    latency = "latency"
    emissions = "emissions"


class StorageKind(str, enum.Enum):
    azure = "azure"
    s3 = "s3"


def cli(
        source_model_uri: str = typer.Argument(..., help=ARG_SOURCE_MODEL_URI_HELP),
        destination_base_uri: str = typer.Argument(..., help=ARG_DESTINATION_BASE_URI_HELP),
        optimized_model_descriptor_uri: str = typer.Argument(..., help=ARG_OPTIMIZED_MODEL_DESCRIPTOR_URI_HELP),
        destination_storage_kind: StorageKind = typer.Argument(..., case_sensitive=False),
        target: OptimizationTarget = typer.Argument(..., case_sensitive=False, help=ARG_TARGET_HELP),
        debug: bool = False
):
    setup_logging(debug)

    logger.info(f"Downloading model from {source_model_uri}")
    time.sleep(2)

    logger.info(f"Optimizing model with nebullvm 🚀")
    time.sleep(5)

    logger.info(f"Uploading optimized model to {destination_base_uri}")
    time.sleep(10)

    logger.info(f'Selecting best hardware for optimized model - target is "{target}"')
    time.sleep(1)

    logger.info(f"Uploading optimized model descriptor to {optimized_model_descriptor_uri}")
    time.sleep(1)
