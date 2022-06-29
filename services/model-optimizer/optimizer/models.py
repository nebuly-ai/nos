from dataclasses import dataclass
import enum


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
    pass
