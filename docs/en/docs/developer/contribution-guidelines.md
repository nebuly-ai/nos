# Contribution guidelines

## How to submit an issue
Did you spot a bug? Did you come up with a cool idea that you think should be implemented? Well, GitHub issues are the best way to let us know!

We don't have a strict policy on issue generation: just use a meaningful title and specify the problem or your proposal in the first problem comment. Then, you can use GitHub labels to let us know what kind of proposal you are making, for example bug if you are reporting a new bug or enhancement if you are proposing a library improvement


## How to contribute to an issue
We are always delighted to welcome other people to the contributor section!
We are looking forward to welcoming you to the community, but before you rush off and write 1000 lines of code, please take a few minutes to read our tips for contributing to the library.

If it's one of your first contributions, check the tag good first issue 🏁

- Please **fork the library** instead of pulling it and creating a new branch.
- **Work on your fork** and work on your branch. Do not hesitate to ask questions by commenting on the issue or asking in the community chats.
- Open a pull request when you think the problem has been solved.
- In the pull request specify which problems it is solving/closing. For instance, if the pull request solves problem #1, the comment should be `Closes #1`.
- The title of the pull request must be meaningful and self-explanatory.



### Coding style
We use [golangci-lint](https://golangci-lint.run/) to enforce a consistent coding style. You can run the linter by using the following target:
```shell
make lint
```

### License
All the source code files requires a license header. You can add automatically add it to new files by running:
```shell
make license-fix
```

