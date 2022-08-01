from typer.testing import CliRunner
import typer
import unittest
from optimizer.cli import cli


class TestCli(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.runner = CliRunner()
        app = typer.Typer()
        app.command()(cli)
        cls.app = app

    def test_invoke_without_args(self):
        result = self.runner.invoke(self.app)
        self.assertEqual(2, result.exit_code)
