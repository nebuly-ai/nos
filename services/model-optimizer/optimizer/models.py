import enum
from dataclasses import dataclass
from typing import List


class OptimizationTarget(str, enum.Enum):
    cost = "cost"
    latency = "latency"
    emissions = "emissions"


class StorageKind(str, enum.Enum):
    azure = "azure"
    s3 = "s3"


@dataclass
class SelectedHardware:
    name: str
    priority: int


@dataclass
class ModelDescriptor:
    model_uri: str
    selected_hardware: List[SelectedHardware]
