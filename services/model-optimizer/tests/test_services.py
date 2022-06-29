import pathlib
import tempfile
import unittest
from unittest.mock import patch, MagicMock

from optimizer import services


class TestModelOptimizer(unittest.TestCase):
    def setUp(self) -> None:
        self.__tmp_dir = tempfile.TemporaryDirectory()
        self.working_dir = pathlib.Path(self.__tmp_dir.name)

    def tearDown(self) -> None:
        self.__tmp_dir.cleanup()

    @patch("optimizer.services.requests")
    def test_optimize_model_return_path_included_in_working_dir(self, mocked_requests):
        mocked_resp = MagicMock()
        mocked_resp.content = "foo".encode()
        mocked_requests.get.return_value = mocked_resp

        optimizer = services.ModelOptimizer(self.working_dir)
        optimized_model_path = optimizer.optimize_model("https://foo.bar")
        self.assertIsNotNone(optimized_model_path)
        self.assertTrue(optimized_model_path.exists())
        self.assertTrue(optimized_model_path.is_file())
        self.assertTrue(self.working_dir.is_symlink())
        self.assertTrue(self.working_dir in optimized_model_path.parents)
