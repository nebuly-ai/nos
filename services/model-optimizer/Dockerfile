FROM python:3.8-slim-buster as builder

COPY pyproject.toml .
COPY poetry.lock .
RUN pip3 install poetry
RUN poetry export -f requirements.txt --without-hashes > requirements.txt


FROM python:3.8-slim-buster

WORKDIR /app

COPY --from=builder requirements.txt .
RUN pip3 install -r requirements.txt
COPY optimizer optimizer

ENTRYPOINT [ "python3", "optimizer"]
