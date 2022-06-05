# Nebulgym

Nebulgym is the library for speeding up the training of your AI models up to 10x without changing a single line of code in your training setup.

## Installation
Installing nebulgym is super easy! You can either
* Install from source
* Install from pypi (pip).


We highly suggest to install nebugym on a fresh environment. You can create and manage your environment using conda or another virtual environment managing application. We tested our installation with `venv` produced with `conda`.

### Install from source code
1. Clone nebulgym repository on your local machine
```bash
git clone https://github.com/nebuly-ai/nebulgym
```
2. enter the repo and run the setup.py file
```bash
cd nebulgym && python setup.py install
```

### Install with Pip
Run
```bash
pip install nebulgym
```

## Get started

Nebulgym aims at accelerating your AI training speeding up the way your AI model is mapped to the hardware. In other words, nebulgym accelerates the computation of your forward and backward passes, and the way your data are loaded and stored in order to get a final run time much faster than before.

For getting an astonishing speed up you can simply use the nebugym annotations while defining your AI model. In the snippet below we show an example of how the nebugym annotations can be used for speeding up both the model computing and the data-loading process. 

```python
from typing import List, Callable

import torch
from torch.utils.data import Dataset

from nebulgym.annotations.torch_annotations import patch_torch_module, patch_dataset

# The model takes as input an image of resolution 224x224
@patch_torch_module(patch_backprop=True, backends=["PYTORCH"])
class CustomModel(torch.nn.Module):
    def __init__(self):
        super().__init__()
        self._avg_pool = torch.nn.AvgPool2d(4)
        self._linear = torch.nn.Linear(3136, 1024)
        self._relu = torch.nn.ReLU()
        self._linears = torch.nn.Sequential(
            torch.nn.BatchNorm1d(1024),
            torch.nn.Linear(1024, 2048),
            torch.nn.ReLU(),
            torch.nn.BatchNorm1d(2048),
            torch.nn.Linear(2048, 1024),
            torch.nn.ReLU(),
            torch.nn.BatchNorm1d(1024),
            torch.nn.Linear(1024, 512),
            torch.nn.ReLU(),
            torch.nn.BatchNorm1d(512),
            torch.nn.Linear(512, 256),
            torch.nn.ReLU(),
            torch.nn.Linear(256, 2),
        )

    def forward(self, x):
        x = self._avg_pool(x).mean(dim=-3).view(-1, 3136)
        x = self._relu(self._linear(x))
        return self._linears(x)


@patch_dataset(preloaded_data=20, max_memory_size=None)
class CustomDataset(Dataset):
    def __init__(self, img_paths: List[str], labelling_func: Callable, reading_func: Callable):
        self._images = img_paths
        self._labelling_func = labelling_func
        self._reading_func = reading_func

    def __getitem__(self, item):
        img_path = self._images[item]
        label = self._labelling_func(img_path)
        input_tensor = self._reading_func(img_path)
        return input_tensor, label

    def __len__(self):
        return len(self._images)
```

In the example above we just defined a model and a dataset in the usual `pytorch` syntax and we have annotated the two classes using the nebulgym annotations: `patch_torch_module` and `patch_dataset`. The two classes can then be used exactly as before, without caring about all the optimization techniques that nebulgym is running on the backend.

## Annotations: The future of AI optimization
With nebulgym we didn't want to create the umpteenth framework for AI training. On the contrary we wanted to leverage on the amazing libraries already on the market and produce an easy to use "plugin" for accelerating the model computation without forcing the user to adopt one training strategy or the other. So, we decided to use the less invasive API we were able to think about: annotations. The idea is that annotating the model or/and the dataset the user selects the object that nebulgym will accelerate without impacting the user code. 

At the current stage, nebulgym supports two different type of annotations: dataset and model annotations.

* `@patch_dataset`: it is the annotation for accelerating the data loading phase. It can take as extra arguments the maximum memory that the data-loading phase can access and the number of "parallel" workers allowed to pre-load the data while training the model.
* `patch_torch_module`: the annotation can be used for accelerating the model computation. It accepts as extra argument the desired backend (see backend section in the following) and the boolean parameter `patch_backprop`. The latest can be used for further accelerating the model modifying the way backprop is computed as well. The backprop optimization can lead to amazing speed-ups without negatively affecting the training itself. However, since it slightly changes the backprop computation we prefer to leave to the final user the decision either adopting or not the technique.

### Supported Backends
At the current stage Nebulgym supports three different backends:
* RAMMER: Compiler developed in [this paper](https://www.usenix.org/system/files/osdi20-ma.pdf) and which can be used on Nvidia GPUs.
* ONNXRUNTIME: training API leveraging on some techniques developed for inference optimization. It currently supports only Nvidia GPUs.
* PYTORCH: Default backend running on pure pytorch.

Note that if the backends selected by the user fails when instantiated the running backend will be switch automatically to pytorch as default. 
