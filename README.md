Star catcher
============

```{bash}
# Ireland
export AWS_REGION=eu-west-1

# Run app with
AWS_PROFILE=starcatcher myapp
```

* Use dynamo DB (free tier). Each document describes an hour, i.e. key 201804051300 describes state for that hour (or when the execution was run).
* Each document should contain a dictionary of repos that are being watched
* each key should contain required scrape data, i.e. `stargazer_count` etc.

## Dependencies

Uses [Dep](https://github.com/golang/dep) for dependency management.

```{bash}
brew install dep
```

