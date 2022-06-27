import enum
import time

import typer
from loguru import logger
import sys

ARG_SOURCE_MODEL_URI_HELP = "URI pointing to the model to optimize"
ARG_OPTIMIZED_MODEL_DESTINATION_URI_HELP = "URI pointing to the destination to which upload the optimized model"
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


def cli(
        source_model_uri: str = typer.Argument(..., help=ARG_SOURCE_MODEL_URI_HELP),
        optimized_model_destination_uri: str = typer.Argument(..., help=ARG_OPTIMIZED_MODEL_DESTINATION_URI_HELP),
        target: OptimizationTarget = typer.Argument(..., case_sensitive=False, help=ARG_TARGET_HELP),
        debug: bool = False
):
    setup_logging(debug)

    logger.info(f"Downloading model from {source_model_uri}")
    time.sleep(2)

    logger.info(f"Optimizing model with nebullvm 🚀")
    time.sleep(5)

    logger.info(f"Uploading optimized model to {optimized_model_destination_uri}")
    time.sleep(10)

    logger.info(f'Selecting best hardware for optimized model - target is "{target}"')
    time.sleep(1)

    logger.info(f"Uploading model-info.json to {optimized_model_destination_uri}")
    time.sleep(1)


