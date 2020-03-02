This is a project that can be used by GitHub projects to show a status badges
on the README file. Many CIs have a dedicated pipeline for the master branch
and that pipeline usually has a URL you can access to display the current status.
Sometimes though, the same pipeline might build and test many branches or PRs
and there is no unique URL that can be used as a badge. Still your CI can talk
to the GitHub API and update the commit status of any commit.

This project assumes your CI is updating the commit status of the configured branch.
E.g. Your CI updates the status of every commit that becomes the tip of your master branch.

It polls the GitHub API and fetches the commit status of the tip of the
configured branch (e.g. master)

It also exposes and endpoint to get a badge icon for any context for that commit.
The endpoint will serve a badge icon (pending, failed, success) that can be used
as a badge on the project's README file.

## Build

```
$ make
```

## Run

Put the required env variables in an `.env` file and run the docker image:

Example .env file:

```
GITHUB_REPO=jimmykarily/github_branch_status
GITHUB_TOKEN=your_github_token_here
```

and run:

```
$ docker run --env-file .env -p 8080:8080  -it jimmykarily/github_branch_status:latest
```

Now visit `127.0.0.1:8080?context=some_context` in your browser and enjoy your badge!
