from torch.utils.data import Dataset

from nebulgym.data.base import BaseDataset

try:
    from bagua.torch_api.contrib.cached_dataset import CachedDataset
except ImportError:
    from .nebuly_dataset import NebulDataset

    CachedDataset = NebulDataset


class BaguaDataset(BaseDataset):
    """Wrapper around bagua's cached dataset. All the Storage Optimization
    techniques are delegated to Bagua's dataset.
    """

    def __init__(self, input_data: Dataset):
        input_data = CachedDataset(input_data)
        super().__init__(input_data)

    def __len__(self):
        return len(self._data)

    def __getitem__(self, item):
        return self._data[item]
