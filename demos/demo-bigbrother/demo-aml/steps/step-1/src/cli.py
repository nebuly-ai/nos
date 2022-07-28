import datetime
import logging
from argparse import ArgumentParser

current_date_time = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
logging.basicConfig(
    level=logging.INFO,
    format="[%(levelname)s - %(asctime)s] %(message)s",
    handlers=[
        logging.StreamHandler(),
    ]
)
log = logging.getLogger(__name__)


def cli():
    arg_parser = ArgumentParser("training")
    arg_parser.add_argument(
        "model_path",
        type=str,
        help="Path to the model to use"
    )
    return arg_parser.parse_args()


def train():
    logging.info("Step 1 - Mocked training")


def main():
    args = cli()
    logging.info("Command-line arguments: %s", args)
    train()
