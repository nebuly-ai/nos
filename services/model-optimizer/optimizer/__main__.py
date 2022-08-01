import typer

from optimizer.cli import cli


def main():
    typer.run(cli)


if __name__ == "__main__":
    main()
