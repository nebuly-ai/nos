try:
    import cli
except ImportError:
    from . import cli

if __name__ == "__main__":
    cli.main()
