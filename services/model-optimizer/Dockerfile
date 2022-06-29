FROM python:3.8-slim-buster as builder

COPY pyproject.toml .
COPY poetry.lock .
RUN pip3 install poetry
RUN poetry export -f requirements.txt --without-hashes > requirements.txt


FROM python:3.8-slim-buster

WORKDIR /app

RUN useradd -m -u 1000 worker
USER 1000

COPY --from=builder requirements.txt .
RUN pip3 install --user -r requirements.txt
COPY optimizer optimizer
COPY pyproject.toml .
RUN pip3 install --user .

ENTRYPOINT [ "python3", "optimizer"]
