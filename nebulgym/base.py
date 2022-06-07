from enum import Enum


class NebulgymBackend(str, Enum):
    PYTORCH = "PYTORCH"
    ONNXRUNTIME = "ONNXRUNTIME"
    RAMMER = "RAMMER"
