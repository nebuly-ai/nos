from typing import Callable


def without_decorator(func: Callable, decorator: Callable) -> Callable:
    """Remove from the function provided as first argument the decorator provided as second argument, if present
    Args:
        func: Callable
            The function from which the decorator has to be removed
        decorator: Callable
            The decorator to remove

    Returns: Callable

    """
    # TODO: proper implementation
    if getattr(func, "__wrapped__") is not None:
        return getattr(func, "__wrapped__")
