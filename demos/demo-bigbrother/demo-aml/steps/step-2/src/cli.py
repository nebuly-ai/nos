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
    arg_parser = ArgumentParser("Test model")
    arg_parser.add_argument(
        "foo",
        type=str,
    )
    return arg_parser.parse_args()


def execute():
    logging.info("Step 2 - Mocked test model executed")


def main():
    args = cli()
    logging.info("Command-line arguments: %s", args)
    execute()
