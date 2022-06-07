from abc import ABC, abstractmethod
from typing import Any

from torch.utils.data import Dataset


class BaseDataset(Dataset, ABC):
    """Base class for dataset.

    Attributes:
        input_data (Any): Input data to be loaded
    """

    def __init__(self, input_data: Any):
        self._data = input_data

    @abstractmethod
    def __len__(self):
        raise NotImplementedError

    @abstractmethod
    def __getitem__(self, item):
        raise NotImplementedError
