import contextlib
import warnings


def suppress_warnings(target_func):
    def _function(*args, **kwargs):
        with warnings.catch_warnings():
            warnings.simplefilter("ignore")
            return target_func(*args, **kwargs)

    return _function


def suppress_stdout(target_func):
    def _function(*args, **kwargs):
        with contextlib.redirect_stdout(None):
            return target_func(*args, **kwargs)

    return _function
