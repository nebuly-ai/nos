# Demo MLRUN

Prerequisites:
* docker-compose
* Python 3.8

## How to run
0. First, if you're using MacOS, install cryptography lib as described [here](https://cryptography.io/en/latest/installation/#building-cryptography-on-macos)
1. Install requirements
`pip install -r requirements`
2. Start MLRun
`make install`

### Does Nuclio support node selection for functions?
Yes, see:
- https://github.com/nuclio/nuclio/blob/development/docs/reference/function-configuration/function-configuration-reference.md
- https://nuclio.io/docs/latest/tasks/deploying-functions/
